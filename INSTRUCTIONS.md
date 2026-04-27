# Extending the Agent

This guide explains how to extend the agent in four common axes:

1. **Conversation** — how messages flow in/out and how to customize them.
2. **Memory** — how to persist or summarize history beyond a single `Run`.
3. **Tools** — how to add new capabilities the model can call.
4. **The agent loop** — how to change orchestration behavior in `Agent.Run`.

The architecture is hexagonal (see `AGENTS.md`):

```
cmd/agent (composition root)
   └── internal/app          (Agent loop — pure orchestration)
         └── internal/ports  (ChatCompleter, Tool, ToolRegistry)
               └── internal/domain (Message, ToolCall, ToolSchema)
internal/adapters            (concrete implementations of ports)
```

Dependencies always point **inward**. When extending, add concrete types in
`internal/adapters/...` and wire them in `cmd/agent/main.go`. Only change
`internal/app` or `internal/ports` when the contract itself needs to evolve.

---

## 1. Extending Conversation

The conversation is just `[]domain.Message` (see
`internal/domain/message.go:29`). The agent does **not** own history — the
caller passes it in to `Agent.Run` and receives the full transcript back
(`internal/app/agent.go:65`).

### Add a new message role or field
Edit `internal/domain/message.go`:
- New role constants: add to the `Role` block at line 10.
- New per-message metadata (e.g. `Name`, `Timestamp`): add to the `Message`
  struct and provide a constructor (e.g. `User`, `Assistant`).
- Update the OpenAI adapter at `internal/adapters/openaichat/client.go` to
  serialize the new field on the wire.

### Customize the system / user prompt
Prompts are built in `cmd/agent/main.go:88`. To template prompts, build them
in the composition root before calling `agent.Run`. Keep template logic out
of `internal/app`.

### Stream assistant output
The current `ChatCompleter` returns one `domain.Message`. For streaming:
- Add a streaming method to `ports.ChatCompleter` (or a sibling port,
  e.g. `ChatStreamer`).
- Implement it in `internal/adapters/openaichat`.
- Add `Agent.RunStream(ctx, in, out chan<- domain.Chunk)` in
  `internal/app/agent.go`, mirroring `Run` but emitting partial deltas.

### Multi-turn / interactive REPL
`Run` is single-shot. For a REPL:
- Loop in `cmd/agent/main.go`, reading user input from stdin.
- Append the new `domain.User(...)` to the transcript returned from the
  previous `Run` and call `Run` again.
- This keeps `Agent` stateless and the history owned by the shell.

---

## 2. Extending Memory

The agent itself has no memory. Add it as an adapter behind a new port so
the application core stays pure.

### Step 1 — define a port
Create `internal/ports/memory.go`:

```go
type Memory interface {
    Load(ctx context.Context, sessionID string) ([]domain.Message, error)
    Save(ctx context.Context, sessionID string, msgs []domain.Message) error
}
```

### Step 2 — implement adapters
Add implementations under `internal/adapters/memory/`:
- `inmemory/` — `map[string][]domain.Message` for tests.
- `file/` — JSON file per session.
- `sqlite/` — durable storage.

Each is a separate Go package, one mock per interface in
`internal/testutil/` per `AGENTS.md`.

### Step 3 — wire it into the agent
Two options, pick based on intent:

**A. Caller-managed (preferred — keeps `Agent` stateless).** The
composition root in `cmd/agent/main.go` calls `mem.Load`, prepends to the
prompt, runs the agent, then calls `mem.Save` on the returned transcript.

**B. Agent-managed.** Add an option to `internal/app/agent.go`:

```go
func WithMemory(m ports.Memory, sessionID string) Option { ... }
```

In `Run`, prepend `mem.Load(...)` results before the first inference call
and `mem.Save(...)` before returning. Trade-off: the agent now has a side
effect that must be mocked in tests.

### Step 4 — summarization / windowing
For long histories, add a `MemoryCompactor` port:

```go
type MemoryCompactor interface {
    Compact(ctx context.Context, msgs []domain.Message) ([]domain.Message, error)
}
```

Call it inside the loop when `len(msgs) > threshold`. A common
implementation issues a "summarize the conversation so far" call through
the same `ChatCompleter` and replaces older messages with the summary.

---

## 3. Extending Tools

Tools implement `ports.Tool` (`internal/ports/tool.go:10`). The clock tool
at `internal/adapters/tools/clock/clock.go` is the canonical example.

### Add a new tool
1. Create a package `internal/adapters/tools/<name>/`.
2. Implement two methods:
   - `Schema() domain.ToolSchema` — JSON-Schema parameters; this is what
     the model sees.
   - `Execute(ctx, args map[string]any) (string, error)` — pure logic, no
     globals. Inject clocks, HTTP clients, etc. via functional options
     (see `clock.WithNow` at `internal/adapters/tools/clock/clock.go:26`).
3. Write a table-driven test next to it (`<name>_test.go`).
4. Register it in `cmd/agent/main.go:76`:
   ```go
   tools := registry.New(clock.New(), weather.New(httpClient), ...)
   ```

### Argument decoding
Arguments arrive as `map[string]any` (decoded from JSON in
`internal/app/agent.go:123`). For complex schemas, define a typed struct
inside the tool package and `json.Unmarshal` the raw map yourself for
strict validation. Return descriptive errors — they are surfaced to the
model verbatim as `error: ...` (`internal/app/agent.go:114`), letting the
model self-correct.

### Tool concurrency
Tool calls run **sequentially** in `runToolCalls`
(`internal/app/agent.go:97`). To parallelize independent calls:
- Use `errgroup.Group` over `calls` and collect results in a fixed-size
  slice indexed by call order.
- Preserve order in the returned `[]domain.Message` so tool-result IDs
  align with the model's expectations.

### Tool authorization / guardrails
Wrap a tool with a decorator that implements `ports.Tool`:

```go
type confirming struct{ inner ports.Tool; confirm func(string) bool }
func (c *confirming) Schema() domain.ToolSchema { return c.inner.Schema() }
func (c *confirming) Execute(ctx context.Context, args map[string]any) (string, error) {
    if !c.confirm(c.inner.Schema().Name) { return "denied by user", nil }
    return c.inner.Execute(ctx, args)
}
```

Register the wrapped tool. The agent core never needs to change.

---

## 4. Modifying the Agent Loop

The loop lives entirely in `Agent.Run` (`internal/app/agent.go:65`). Keep
it small; push policy out via options.

### Add a new option
Follow the existing `Option func(*Agent)` pattern at line 28. Examples:

- `WithRetry(n int, backoff time.Duration)` — wrap the `chat.Complete`
  call in a retry helper.
- `WithToolTimeout(d time.Duration)` — wrap `ctx` with
  `context.WithTimeout` before each `tool.Execute`.
- `WithStopWhen(fn func([]domain.Message) bool)` — early-exit predicate
  evaluated after each iteration.
- `WithObserver(o ports.LoopObserver)` — emit structured events
  (richer than the current `tracef` at line 134).

Constructor wiring happens in `NewAgent` at line 44.

### Hooks / middleware
For cross-cutting behavior (logging, metrics, redaction), introduce a
slice of middleware:

```go
type CompleterMiddleware func(ports.ChatCompleter) ports.ChatCompleter
func WithCompleterMiddleware(mw ...CompleterMiddleware) Option { ... }
```

Apply them in `NewAgent` so `a.chat` is the wrapped completer. Same
pattern works for tools.

### Changing iteration semantics
The loop currently:
1. Calls the model.
2. Returns if no tool calls.
3. Otherwise executes tools and repeats, up to `maxIts`
   (`internal/app/agent.go:68`).

Common modifications:
- **Plan / act / reflect**: split into phases by alternating system
  messages or by a `Phase` field on the agent.
- **Branching**: if you need parallel speculative runs, copy `msgs` per
  branch — the slice is already cloned defensively at line 66.
- **Budget enforcement**: track tokens via the adapter and stop when a
  cap is hit. Add a `TokenCounter` port; do not put SDK token math in
  `internal/app`.

### Error policy
Today, tool execution errors are returned to the model as content; only
unknown tools or arg-decode failures abort the run
(`internal/app/agent.go:103`). To make tool errors fatal, return early
from `runToolCalls`. To classify errors (retryable vs. fatal), introduce
a small `ToolError` type in `internal/domain` and let
`Agent.runToolCalls` branch on it.

---

## Checklist when extending

Per `AGENTS.md`:

- [ ] New code lives in `internal/adapters/...` unless it's a contract.
- [ ] New contract goes in `internal/ports/` with a mock in
      `internal/testutil/`.
- [ ] `cmd/agent/main.go` is the only place that wires concrete types.
- [ ] Tests are table-driven and use `t.Cleanup` for teardown.
- [ ] `make fmt && make lint && make test` is green before committing.

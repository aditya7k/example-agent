// Package app contains the application-layer agent loop. It depends only
// on ports, never on concrete adapters or third-party SDKs.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"example.com/gh-models-agent/internal/domain"
	"example.com/gh-models-agent/internal/ports"
)

// DefaultMaxIterations bounds the tool-calling loop.
const DefaultMaxIterations = 5

// Agent orchestrates a chat-completions tool-calling loop.
type Agent struct {
	chat   ports.ChatCompleter
	tools  ports.ToolRegistry
	model  string
	maxIts int
	trace  io.Writer // optional; nil disables trace output
}

// Option configures an Agent.
type Option func(*Agent)

// WithMaxIterations caps the number of model turns. Must be > 0.
func WithMaxIterations(n int) Option {
	return func(a *Agent) {
		if n > 0 {
			a.maxIts = n
		}
	}
}

// WithTrace writes one line per tool call/result to w. Pass nil to disable.
func WithTrace(w io.Writer) Option { return func(a *Agent) { a.trace = w } }

// NewAgent wires the agent's collaborators. Returns an error if any
// required dependency is missing.
func NewAgent(chat ports.ChatCompleter, tools ports.ToolRegistry, model string, opts ...Option) (*Agent, error) {
	if chat == nil {
		return nil, fmt.Errorf("app: chat completer is required")
	}
	if tools == nil {
		return nil, fmt.Errorf("app: tool registry is required")
	}
	if model == "" {
		return nil, fmt.Errorf("app: model is required")
	}
	a := &Agent{chat: chat, tools: tools, model: model, maxIts: DefaultMaxIterations}
	for _, o := range opts {
		o(a)
	}
	return a, nil
}

// Run executes the tool-calling loop, returning the final transcript
// (input messages plus all assistant/tool turns produced).
//
// The slice passed in is not mutated; a new slice is returned.
func (a *Agent) Run(ctx context.Context, in []domain.Message) ([]domain.Message, error) {
	msgs := append([]domain.Message(nil), in...)

	for i := 0; i < a.maxIts; i++ {
		assistant, err := a.chat.Complete(ctx, ports.ChatRequest{
			Model:    a.model,
			Messages: msgs,
			Tools:    a.tools.Schemas(),
		})
		if err != nil {
			return nil, fmt.Errorf("inference call failed: %w", err)
		}
		msgs = append(msgs, assistant)

		if len(assistant.ToolCalls) == 0 {
			return msgs, nil
		}

		results, err := a.runToolCalls(ctx, assistant.ToolCalls)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, results...)
	}

	return nil, fmt.Errorf("agent did not converge within %d iterations", a.maxIts)
}

// runToolCalls executes each tool call sequentially and produces tool-role
// messages with the results. Tool errors are reported back to the model as
// content rather than aborting the loop, matching the pattern used by most
// agentic frameworks.
func (a *Agent) runToolCalls(ctx context.Context, calls []domain.ToolCall) ([]domain.Message, error) {
	out := make([]domain.Message, 0, len(calls))
	for _, tc := range calls {
		a.tracef("[tool call] %s(%s)\n", tc.Name, tc.Arguments)

		tool, ok := a.tools.Get(tc.Name)
		if !ok {
			return nil, fmt.Errorf("unknown tool: %s", tc.Name)
		}

		args, err := decodeArgs(tc.Arguments)
		if err != nil {
			return nil, fmt.Errorf("bad tool args for %s: %w", tc.Name, err)
		}

		result, err := tool.Execute(ctx, args)
		if err != nil {
			result = "error: " + err.Error()
		}
		a.tracef("[tool result] %s\n", result)

		out = append(out, domain.ToolResult(tc.ID, result))
	}
	return out, nil
}

func decodeArgs(raw string) (map[string]any, error) {
	if raw == "" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (a *Agent) tracef(format string, args ...any) {
	if a.trace == nil {
		return
	}
	fmt.Fprintf(a.trace, format, args...)
}

// Command agent is the entry point binary. It is a thin shell: it parses
// flags, wires concrete adapters into application services (the
// composition root, per CODING_STYLE.md), and prints output.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"example.com/gh-models-agent/internal/adapters/openaichat"
	"example.com/gh-models-agent/internal/adapters/tools/clock"
	"example.com/gh-models-agent/internal/adapters/tools/registry"
	"example.com/gh-models-agent/internal/app"
	"example.com/gh-models-agent/internal/domain"
)

const (
	defaultModel       = "openai/gpt-4o-mini"
	defaultSessionFile = "session/session1.md"
)

type runOpts struct {
	model       string
	prompt      string
	system      string
	maxIters    int
	trace       bool
	sessionFile string
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var o runOpts

	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Hello-world agent backed by GitHub Models",
		Long: "A small tool-calling agent that talks to the GitHub Models " +
			"OpenAI-compatible inference endpoint. Requires GITHUB_TOKEN.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), o, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cmd.Flags().StringVar(&o.model, "model", defaultModel, "model id from the GitHub Models marketplace")
	cmd.Flags().StringVar(&o.prompt, "prompt", "Say hello, then tell me the current time in Tokyo.", "user prompt")
	cmd.Flags().StringVar(&o.system, "system", "You are a friendly hello-world agent. When asked for the time, use the get_current_time tool.", "system prompt")
	cmd.Flags().IntVar(&o.maxIters, "max-iters", app.DefaultMaxIterations, "maximum number of model turns")
	cmd.Flags().BoolVar(&o.trace, "trace", true, "print tool calls and results")
	cmd.Flags().StringVar(&o.sessionFile, "session", defaultSessionFile, "path to session markdown file (loaded then overwritten)")

	return cmd
}

// run is the composition root: it builds adapters, wires them into the
// application service, executes one conversation, and prints the result.
func run(ctx context.Context, o runOpts, stdout, stderr io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN is not set")
	}

	chat, err := openaichat.New(openaichat.WithAPIKey(token))
	if err != nil {
		return fmt.Errorf("failed to build chat client: %w", err)
	}

	tools := registry.New(clock.New())

	opts := []app.Option{app.WithMaxIterations(o.maxIters)}
	if o.trace {
		opts = append(opts, app.WithTrace(stderr))
	}

	agent, err := app.NewAgent(chat, tools, o.model, opts...)
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	prior, err := loadSession(o.sessionFile)
	if err != nil {
		return fmt.Errorf("failed to load session %s: %w", o.sessionFile, err)
	}

	in := prior
	if len(in) == 0 {
		in = append(in, domain.System(o.system))
	}
	in = append(in, domain.User(o.prompt))

	transcript, err := agent.Run(ctx, in)
	if err != nil {
		return err
	}

	if err := saveSession(o.sessionFile, transcript); err != nil {
		return fmt.Errorf("failed to save session %s: %w", o.sessionFile, err)
	}

	final := transcript[len(transcript)-1]
	if final.Content != "" {
		fmt.Fprintln(stdout, "\n[assistant]")
		fmt.Fprintln(stdout, final.Content)
	}
	return nil
}

// sessionDelimiter separates messages in the markdown session file.
const sessionDelimiter = "\n\n---\n\n"

// loadSession parses a previously persisted session file. Returns an empty
// slice if the file does not exist.
func loadSession(path string) ([]domain.Message, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var msgs []domain.Message
	for _, block := range strings.Split(string(raw), sessionDelimiter) {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		// Each block: "## <role>\n\n<content>" with optional fenced JSON
		// payload for tool_calls / tool_call_id.
		lines := strings.SplitN(block, "\n", 2)
		if len(lines) < 2 || !strings.HasPrefix(lines[0], "## ") {
			continue
		}
		role := domain.Role(strings.TrimSpace(strings.TrimPrefix(lines[0], "## ")))
		body := strings.TrimSpace(lines[1])

		msg := domain.Message{Role: role}
		if i := strings.Index(body, "\n```json\n"); i >= 0 {
			meta := body[i+len("\n```json\n"):]
			body = strings.TrimSpace(body[:i])
			if end := strings.Index(meta, "\n```"); end >= 0 {
				meta = meta[:end]
			}
			var m struct {
				ToolCalls  []domain.ToolCall `json:"tool_calls,omitempty"`
				ToolCallID string            `json:"tool_call_id,omitempty"`
			}
			if err := json.Unmarshal([]byte(meta), &m); err != nil {
				return nil, fmt.Errorf("decode metadata: %w", err)
			}
			msg.ToolCalls = m.ToolCalls
			msg.ToolCallID = m.ToolCallID
		}
		msg.Content = body
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

// saveSession writes the full transcript to path as a markdown document.
func saveSession(path string, msgs []domain.Message) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("# Agent Session\n")
	for _, m := range msgs {
		b.WriteString(sessionDelimiter)
		fmt.Fprintf(&b, "## %s\n\n", m.Role)
		if m.Content != "" {
			b.WriteString(m.Content)
			b.WriteString("\n")
		}
		if len(m.ToolCalls) > 0 || m.ToolCallID != "" {
			meta := struct {
				ToolCalls  []domain.ToolCall `json:"tool_calls,omitempty"`
				ToolCallID string            `json:"tool_call_id,omitempty"`
			}{ToolCalls: m.ToolCalls, ToolCallID: m.ToolCallID}
			j, err := json.MarshalIndent(meta, "", "  ")
			if err != nil {
				return err
			}
			b.WriteString("\n```json\n")
			b.Write(j)
			b.WriteString("\n```\n")
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

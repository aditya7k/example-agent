// Package ports defines the interfaces the application core uses to talk
// to the outside world. Adapters in internal/adapters implement them.
package ports

import (
	"context"

	"example.com/gh-models-agent/internal/domain"
)

// ChatRequest is the input to a single inference call.
type ChatRequest struct {
	Model    string
	Messages []domain.Message
	Tools    []domain.ToolSchema
}

// ChatCompleter is the port the agent uses to obtain the next assistant
// message from a language model. Implementations are expected to be
// stateless w.r.t. the conversation — the caller owns the message history.
type ChatCompleter interface {
	Complete(ctx context.Context, req ChatRequest) (domain.Message, error)
}

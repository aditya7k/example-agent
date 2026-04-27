package ports

import (
	"context"

	"example.com/gh-models-agent/internal/domain"
)

// Tool is a single capability the agent can invoke on behalf of the model.
type Tool interface {
	Schema() domain.ToolSchema
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// ToolRegistry resolves tools by name and exposes the full set of schemas
// to advertise to the model.
type ToolRegistry interface {
	Get(name string) (Tool, bool)
	Schemas() []domain.ToolSchema
}

// Package clock provides a Tool that reports the current time in a given
// IANA timezone. The clock source is injectable to keep the tool pure and
// trivially testable.
package clock

import (
	"context"
	"fmt"
	"time"

	"example.com/gh-models-agent/internal/domain"
)

// ToolName is the canonical name advertised to the model.
const ToolName = "get_current_time"

// Tool implements ports.Tool for the current-time capability.
type Tool struct {
	now func() time.Time
}

// Option configures a Tool.
type Option func(*Tool)

// WithNow overrides the time source. Defaults to time.Now.
func WithNow(now func() time.Time) Option { return func(t *Tool) { t.now = now } }

// New constructs a Tool with optional overrides.
func New(opts ...Option) *Tool {
	t := &Tool{now: time.Now}
	for _, o := range opts {
		o(t)
	}
	return t
}

// Schema returns the model-facing description.
func (t *Tool) Schema() domain.ToolSchema {
	return domain.ToolSchema{
		Name:        ToolName,
		Description: "Return the current time in the given IANA timezone (e.g. 'America/Los_Angeles').",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timezone": map[string]any{
					"type":        "string",
					"description": "IANA timezone name. Defaults to UTC if omitted.",
				},
			},
			"required": []string{},
		},
	}
}

// Execute resolves the timezone and formats the current time as RFC3339.
func (t *Tool) Execute(_ context.Context, args map[string]any) (string, error) {
	tz, _ := args["timezone"].(string)
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", fmt.Errorf("unknown timezone %q: %w", tz, err)
	}
	return t.now().In(loc).Format(time.RFC3339), nil
}

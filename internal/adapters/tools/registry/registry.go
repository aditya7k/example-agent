// Package registry is an in-memory ports.ToolRegistry implementation.
package registry

import (
	"example.com/gh-models-agent/internal/domain"
	"example.com/gh-models-agent/internal/ports"
)

// Registry is an immutable set of tools indexed by name.
type Registry struct {
	tools map[string]ports.Tool
	order []string // preserves insertion order for stable Schemas() output
}

// New constructs a Registry from the supplied tools. Later tools with the
// same name overwrite earlier ones.
func New(tools ...ports.Tool) *Registry {
	r := &Registry{tools: make(map[string]ports.Tool, len(tools))}
	for _, t := range tools {
		name := t.Schema().Name
		if _, exists := r.tools[name]; !exists {
			r.order = append(r.order, name)
		}
		r.tools[name] = t
	}
	return r
}

// Get resolves a tool by name.
func (r *Registry) Get(name string) (ports.Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Schemas returns the schemas of all registered tools in insertion order.
func (r *Registry) Schemas() []domain.ToolSchema {
	out := make([]domain.ToolSchema, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.tools[name].Schema())
	}
	return out
}

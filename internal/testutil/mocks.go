// Package testutil provides shared test doubles for ports interfaces.
//
// Per AGENTS.md: one mock per interface, located here.
package testutil

import (
	"context"

	"github.com/stretchr/testify/mock"

	"example.com/gh-models-agent/internal/domain"
	"example.com/gh-models-agent/internal/ports"
)

// ChatCompleterMock is a testify mock for ports.ChatCompleter.
type ChatCompleterMock struct{ mock.Mock }

// Complete mocks the eponymous port method.
func (m *ChatCompleterMock) Complete(ctx context.Context, req ports.ChatRequest) (domain.Message, error) {
	args := m.Called(ctx, req)
	msg, _ := args.Get(0).(domain.Message)
	return msg, args.Error(1)
}

// ToolMock is a testify mock for ports.Tool.
type ToolMock struct{ mock.Mock }

// Schema mocks the eponymous port method.
func (m *ToolMock) Schema() domain.ToolSchema {
	args := m.Called()
	s, _ := args.Get(0).(domain.ToolSchema)
	return s
}

// Execute mocks the eponymous port method.
func (m *ToolMock) Execute(ctx context.Context, a map[string]any) (string, error) {
	args := m.Called(ctx, a)
	return args.String(0), args.Error(1)
}

// ToolRegistryMock is a testify mock for ports.ToolRegistry.
type ToolRegistryMock struct{ mock.Mock }

// Get mocks the eponymous port method.
func (m *ToolRegistryMock) Get(name string) (ports.Tool, bool) {
	args := m.Called(name)
	t, _ := args.Get(0).(ports.Tool)
	return t, args.Bool(1)
}

// Schemas mocks the eponymous port method.
func (m *ToolRegistryMock) Schemas() []domain.ToolSchema {
	args := m.Called()
	s, _ := args.Get(0).([]domain.ToolSchema)
	return s
}

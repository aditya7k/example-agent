package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"example.com/gh-models-agent/internal/app"
	"example.com/gh-models-agent/internal/domain"
	"example.com/gh-models-agent/internal/ports"
	"example.com/gh-models-agent/internal/testutil"
)

const testModel = "test-model"

// newAgent assembles an Agent with the given mocks and registers cleanup.
func newAgent(t *testing.T, opts ...app.Option) (*app.Agent, *testutil.ChatCompleterMock, *testutil.ToolRegistryMock) {
	t.Helper()
	chat := &testutil.ChatCompleterMock{}
	reg := &testutil.ToolRegistryMock{}
	a, err := app.NewAgent(chat, reg, testModel, opts...)
	require.NoError(t, err)
	t.Cleanup(func() {
		chat.AssertExpectations(t)
		reg.AssertExpectations(t)
	})
	return a, chat, reg
}

func TestNewAgent_Configuration(t *testing.T) {
	tests := []struct {
		name    string
		chat    ports.ChatCompleter
		tools   ports.ToolRegistry
		model   string
		wantErr bool
	}{
		{name: "happy path", chat: &testutil.ChatCompleterMock{}, tools: &testutil.ToolRegistryMock{}, model: testModel},
		{name: "missing chat", tools: &testutil.ToolRegistryMock{}, model: testModel, wantErr: true},
		{name: "missing registry", chat: &testutil.ChatCompleterMock{}, model: testModel, wantErr: true},
		{name: "missing model", chat: &testutil.ChatCompleterMock{}, tools: &testutil.ToolRegistryMock{}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := app.NewAgent(tc.chat, tc.tools, tc.model)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAgent_Run_TerminatesWithoutToolCalls(t *testing.T) {
	a, chat, reg := newAgent(t)

	reg.On("Schemas").Return([]domain.ToolSchema{}).Once()
	chat.On("Complete", mock.Anything, mock.Anything).
		Return(domain.Assistant("hello world"), nil).Once()

	out, err := a.Run(context.Background(), []domain.Message{domain.User("hi")})

	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, domain.RoleUser, out[0].Role)
	assert.Equal(t, "hello world", out[1].Content)
}

func TestAgent_Run_ExecutesToolCallsAndLoops(t *testing.T) {
	a, chat, reg := newAgent(t)

	tool := &testutil.ToolMock{}
	t.Cleanup(func() { tool.AssertExpectations(t) })

	call := domain.ToolCall{ID: "call_1", Name: "echo", Arguments: `{"msg":"hi"}`}

	reg.On("Schemas").Return([]domain.ToolSchema{}).Twice()
	reg.On("Get", "echo").Return(tool, true).Once()
	tool.On("Execute", mock.Anything, map[string]any{"msg": "hi"}).
		Return("hi", nil).Once()

	chat.On("Complete", mock.Anything, mock.Anything).
		Return(domain.Assistant("", call), nil).Once()
	chat.On("Complete", mock.Anything, mock.Anything).
		Return(domain.Assistant("done"), nil).Once()

	out, err := a.Run(context.Background(), []domain.Message{domain.User("go")})

	require.NoError(t, err)
	require.Len(t, out, 4) // user, assistant(toolcall), tool, assistant(final)
	assert.Equal(t, domain.RoleTool, out[2].Role)
	assert.Equal(t, "call_1", out[2].ToolCallID)
	assert.Equal(t, "hi", out[2].Content)
	assert.Equal(t, "done", out[3].Content)
}

func TestAgent_Run_FailsOnUnknownTool(t *testing.T) {
	a, chat, reg := newAgent(t)

	call := domain.ToolCall{ID: "x", Name: "ghost", Arguments: "{}"}
	reg.On("Schemas").Return([]domain.ToolSchema{}).Once()
	reg.On("Get", "ghost").Return(nil, false).Once()
	chat.On("Complete", mock.Anything, mock.Anything).
		Return(domain.Assistant("", call), nil).Once()

	_, err := a.Run(context.Background(), nil)
	require.ErrorContains(t, err, "unknown tool")
}

func TestAgent_Run_PropagatesChatErrors(t *testing.T) {
	a, chat, reg := newAgent(t)

	reg.On("Schemas").Return([]domain.ToolSchema{}).Once()
	chat.On("Complete", mock.Anything, mock.Anything).
		Return(domain.Message{}, errors.New("boom")).Once()

	_, err := a.Run(context.Background(), nil)
	require.ErrorContains(t, err, "boom")
}

func TestAgent_Run_HonorsMaxIterations(t *testing.T) {
	a, chat, reg := newAgent(t, app.WithMaxIterations(2))

	tool := &testutil.ToolMock{}
	t.Cleanup(func() { tool.AssertExpectations(t) })

	call := domain.ToolCall{ID: "loop", Name: "echo", Arguments: "{}"}

	reg.On("Schemas").Return([]domain.ToolSchema{}).Twice()
	reg.On("Get", "echo").Return(tool, true).Twice()
	tool.On("Execute", mock.Anything, mock.Anything).Return("ok", nil).Twice()

	chat.On("Complete", mock.Anything, mock.Anything).
		Return(domain.Assistant("", call), nil).Twice()

	_, err := a.Run(context.Background(), nil)
	require.ErrorContains(t, err, "did not converge")
}

// Tool error from Execute should be reported back to the model, not abort.
func TestAgent_Run_SurfacesToolErrorAsMessage(t *testing.T) {
	a, chat, reg := newAgent(t)

	tool := &testutil.ToolMock{}
	t.Cleanup(func() { tool.AssertExpectations(t) })

	call := domain.ToolCall{ID: "1", Name: "echo", Arguments: "{}"}

	reg.On("Schemas").Return([]domain.ToolSchema{}).Twice()
	reg.On("Get", "echo").Return(tool, true).Once()
	tool.On("Execute", mock.Anything, mock.Anything).
		Return("", errors.New("nope")).Once()

	chat.On("Complete", mock.Anything, mock.Anything).
		Return(domain.Assistant("", call), nil).Once()
	chat.On("Complete", mock.Anything, mock.Anything).
		Return(domain.Assistant("recovered"), nil).Once()

	out, err := a.Run(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, out, 3)
	assert.Equal(t, "error: nope", out[1].Content)
	assert.Equal(t, "recovered", out[2].Content)
}

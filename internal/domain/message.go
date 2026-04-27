// Package domain holds the pure business types of the agent.
//
// It depends on nothing outside the standard library so it can be reasoned
// about, tested, and reused without dragging in adapters or third-party SDKs.
package domain

// Role identifies the speaker of a Message in a conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall is a request from the model to invoke a named tool with JSON
// arguments. The ID is opaque and must be echoed back on the corresponding
// tool result message so the model can correlate calls and responses.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // raw JSON
}

// Message is a single turn in the conversation.
//
// The zero value is not meaningful; use the constructors below.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall // assistant turns only
	ToolCallID string     // tool turns only — id of the call this responds to
}

// System builds a system-role message.
func System(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

// User builds a user-role message.
func User(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

// Assistant builds an assistant-role message, optionally with tool calls.
func Assistant(content string, calls ...ToolCall) Message {
	return Message{Role: RoleAssistant, Content: content, ToolCalls: calls}
}

// ToolResult builds a tool-role message responding to a prior ToolCall.
func ToolResult(callID, content string) Message {
	return Message{Role: RoleTool, Content: content, ToolCallID: callID}
}

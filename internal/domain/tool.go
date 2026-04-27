package domain

// ToolSchema is the model-facing description of a tool.
//
// Parameters follows JSON-Schema as required by the OpenAI tool-calling
// protocol. It is intentionally a generic map so the domain stays free of
// any SDK-specific types.
type ToolSchema struct {
	Name        string
	Description string
	Parameters  map[string]any
}

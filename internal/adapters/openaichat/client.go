// Package openaichat adapts the OpenAI-compatible chat-completions API
// (in particular the GitHub Models inference endpoint) to the
// ports.ChatCompleter port.
//
// All translation between domain types and SDK types happens here so the
// rest of the application remains free of vendor coupling.
package openaichat

import (
	"context"
	"fmt"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"

	"example.com/gh-models-agent/internal/domain"
	"example.com/gh-models-agent/internal/ports"
)

// GitHubModelsBaseURL is the public GitHub Models inference endpoint.
const GitHubModelsBaseURL = "https://models.github.ai/inference"

// Client implements ports.ChatCompleter against an OpenAI-compatible API.
type Client struct {
	api openai.Client
}

// Option configures a Client.
type Option func(*config)

type config struct {
	baseURL string
	apiKey  string
}

// WithBaseURL overrides the API base URL. Defaults to GitHubModelsBaseURL.
func WithBaseURL(u string) Option { return func(c *config) { c.baseURL = u } }

// WithAPIKey sets the bearer token used to authenticate requests.
func WithAPIKey(k string) Option { return func(c *config) { c.apiKey = k } }

// New constructs a Client. An API key is required; the base URL defaults to
// the GitHub Models endpoint.
func New(opts ...Option) (*Client, error) {
	cfg := config{baseURL: GitHubModelsBaseURL}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.apiKey == "" {
		return nil, fmt.Errorf("openaichat: api key is required")
	}
	api := openai.NewClient(
		option.WithBaseURL(cfg.baseURL),
		option.WithAPIKey(cfg.apiKey),
	)
	return &Client{api: api}, nil
}

// Complete sends one chat-completions request and returns the assistant
// message translated back into the domain type.
func (c *Client) Complete(ctx context.Context, req ports.ChatRequest) (domain.Message, error) {
	params := openai.ChatCompletionNewParams{
		Model:    req.Model,
		Messages: toSDKMessages(req.Messages),
		Tools:    toSDKTools(req.Tools),
	}

	resp, err := c.api.Chat.Completions.New(ctx, params)
	if err != nil {
		return domain.Message{}, fmt.Errorf("openaichat: completion failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return domain.Message{}, fmt.Errorf("openaichat: empty choices in response")
	}
	return fromSDKMessage(resp.Choices[0].Message), nil
}

// --- translation helpers (pure) ----------------------------------------------

func toSDKMessages(msgs []domain.Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case domain.RoleSystem:
			out = append(out, openai.SystemMessage(m.Content))
		case domain.RoleUser:
			out = append(out, openai.UserMessage(m.Content))
		case domain.RoleTool:
			out = append(out, openai.ToolMessage(m.Content, m.ToolCallID))
		case domain.RoleAssistant:
			out = append(out, assistantParam(m))
		}
	}
	return out
}

func assistantParam(m domain.Message) openai.ChatCompletionMessageParamUnion {
	asst := openai.ChatCompletionAssistantMessageParam{}
	if m.Content != "" {
		asst.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
			OfString: openai.String(m.Content),
		}
	}
	if len(m.ToolCalls) > 0 {
		calls := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			fn := openai.ChatCompletionMessageFunctionToolCallParam{
				ID: tc.ID,
				Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			}
			calls = append(calls, openai.ChatCompletionMessageToolCallUnionParam{OfFunction: &fn})
		}
		asst.ToolCalls = calls
	}
	return openai.ChatCompletionMessageParamUnion{OfAssistant: &asst}
}

func toSDKTools(tools []domain.ToolSchema) []openai.ChatCompletionToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		out = append(out, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  shared.FunctionParameters(t.Parameters),
		}))
	}
	return out
}

func fromSDKMessage(m openai.ChatCompletionMessage) domain.Message {
	out := domain.Message{
		Role:    domain.RoleAssistant,
		Content: m.Content,
	}
	if len(m.ToolCalls) > 0 {
		out.ToolCalls = make([]domain.ToolCall, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			out.ToolCalls = append(out.ToolCalls, domain.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}
	return out
}

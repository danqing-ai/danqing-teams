package port

import (
	"context"
	"danqing-teams/core/domain"
)

type ChatMessage struct {
	Role       string
	Content    string
	ToolCalls  []ChatToolCall
	ToolCallID string
	Name       string
}

type ChatToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

type LLMChatRequest struct {
	Model      string
	Messages   []ChatMessage
	Tools      []domain.ToolSchema
	ToolChoice string
	GenParams  *ModelGenParams // optional; nil = provider defaults
}

// ModelGenParams carries per-model generation parameters.
// Zero values mean "use provider default" (do not send to API).
type ModelGenParams struct {
	MaxTokens          int
	Temperature        float64
	TopP               float64
	FrequencyPenalty   float64
	PresencePenalty    float64
	Stop               []string
	ThinkingMode       string         // "adaptive" or "enabled"; for Anthropic thinking
	EffortBudgetTokens map[string]int // effort → budget_tokens mapping
}

type LLMChatResponse struct {
	Content          string
	ReasoningContent string // thinking/reasoning trace from reasoning models
	ToolCalls        []ChatToolCall
	Usage            *LLMUsage
	Done             bool
}

type LLMUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type LLMProvider interface {
	Chat(ctx context.Context, req LLMChatRequest) (LLMChatResponse, error)
}

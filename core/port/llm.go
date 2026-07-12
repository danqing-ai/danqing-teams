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
	Model    string
	Messages []ChatMessage
	Tools    []domain.ToolSchema
	ToolChoice string
}

type LLMChatResponse struct {
	Content   string
	ToolCalls []ChatToolCall
	Usage     *LLMUsage
	Done      bool
}

type LLMUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type LLMProvider interface {
	Chat(ctx context.Context, req LLMChatRequest) (LLMChatResponse, error)
}

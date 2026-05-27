package model

import "context"

// LLMRole identifies which agent persona drives the completion.
type LLMRole string

const (
	LLMRoleController LLMRole = "controller"
	LLMRoleWorker     LLMRole = "worker"
)

type CompletionRequest struct {
	Role    LLMRole
	Prompt  string
	Context map[string]string
}

type CompletionResponse struct {
	Content string
}

// LLMProvider abstracts remote/local LLM backends.
type LLMProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

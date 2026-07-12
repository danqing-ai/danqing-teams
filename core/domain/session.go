package domain

import "time"

type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusArchived  SessionStatus = "archived"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusFailed    SessionStatus = "failed"
)

type Session struct {
	ID        string        `json:"id"`
	Title     string        `json:"title,omitempty"`
	ProjectID string        `json:"projectId,omitempty"`
	AgentID   string        `json:"agentId,omitempty"`
	ModelID   string        `json:"modelId,omitempty"`
	Content   string        `json:"content"`
	Status    SessionStatus `json:"status"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

type CreateSessionRequest struct {
	Content   string `json:"content"`
	AgentID   string `json:"agentId,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	ModelID   string `json:"modelId,omitempty"`
}

type SendMessageRequest struct {
	UserInput string `json:"userInput"`
	AgentID   string `json:"agentId,omitempty"`
	ModelID   string `json:"modelId,omitempty"`
}

type UpdateSessionRequest struct {
	Title     *string        `json:"title,omitempty"`
	ProjectID *string        `json:"projectId,omitempty"`
	Status    *SessionStatus `json:"status,omitempty"`
	ModelID   *string        `json:"modelId,omitempty"`
	AgentID   *string        `json:"agentId,omitempty"`
}

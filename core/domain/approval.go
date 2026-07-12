package domain

import "time"

type Approval struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"sessionId"`
	TurnID      string    `json:"runId"`
	ToolName    string    `json:"toolName"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

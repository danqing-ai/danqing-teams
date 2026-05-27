package dto

import "time"

type MessageRole string

const (
	MessageRoleUser       MessageRole = "user"
	MessageRoleController MessageRole = "controller"
	MessageRoleSystem     MessageRole = "system"
)

type TeamMessage struct {
	ID        string      `json:"id"`
	TeamID    string      `json:"teamId"`
	TaskID    string      `json:"taskId"`
	Role      MessageRole `json:"role"`
	Content   string      `json:"content"`
	CreatedAt time.Time   `json:"createdAt"`
}

type SendTeamMessageRequest struct {
	Content string `json:"content"`
	TaskID  string `json:"taskId,omitempty"`
}

type SendTeamMessageResponse struct {
	Message TeamMessage `json:"message"`
	Task    TeamTask    `json:"task"`
}

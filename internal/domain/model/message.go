package model

import "time"

type MessageRole string

const (
	MessageRoleUser       MessageRole = "user"
	MessageRoleController MessageRole = "controller"
	MessageRoleSystem     MessageRole = "system"
)

type TeamMessage struct {
	ID        string
	TeamID    string
	TaskID    string
	Role      MessageRole
	Content   string
	CreatedAt time.Time
}

type SendTeamMessageRequest struct {
	Content string
	TaskID  string
}

type SendTeamMessageResponse struct {
	Message TeamMessage
	Task    TeamTask
}

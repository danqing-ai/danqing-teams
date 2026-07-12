package domain

type TurnStatus string

const (
	TurnRunning   TurnStatus = "running"
	TurnCompleted TurnStatus = "completed"
	TurnFailed    TurnStatus = "failed"
	TurnCancelled TurnStatus = "cancelled"
	TurnTimeout   TurnStatus = "timeout"
)

type TurnLog struct {
	ID        string     `json:"id"`
	SessionID string     `json:"sessionId"`
	Status    TurnStatus `json:"status"`
	AgentID   string     `json:"agentId"`
	Goal      string     `json:"goal"`
}

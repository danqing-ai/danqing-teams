package domain

type ToolPartStatus string

const (
	ToolPending   ToolPartStatus = "pending"
	ToolRunning   ToolPartStatus = "running"
	ToolCompleted ToolPartStatus = "completed"
	ToolError     ToolPartStatus = "error"
)

type ToolPart struct {
	CallID      string         `json:"callId"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Status      ToolPartStatus `json:"status"`
	Input       map[string]any `json:"input,omitempty"`
	Output      string         `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
}

package permission

import "time"

type ApprovalRecord struct {
	ID          string
	SessionID   string
	ToolName    string
	Summary     string
	Description string
	Status      string
	CreatedAt   time.Time
}

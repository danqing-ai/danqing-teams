package contract

import "time"

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

type ApprovalRequest struct {
	ID            string         `json:"id"`
	TeamID        string         `json:"teamId"`
	TaskID        string         `json:"taskId"`
	RunID         string         `json:"runId"`
	Summary       string         `json:"summary"`
	HighRiskItems []RiskItem     `json:"highRiskItems"`
	Status        ApprovalStatus `json:"status"`
	Comment       string         `json:"comment,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type DecideApprovalRequest struct {
	Comment string `json:"comment,omitempty"`
}

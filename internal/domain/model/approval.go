package model

import "time"

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

type ApprovalRequest struct {
	ID            string
	TeamID        string
	TaskID        string
	RunID         string
	Summary       string
	HighRiskItems []RiskItem
	Status        ApprovalStatus
	Comment       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DecideApprovalRequest struct {
	Comment string
}

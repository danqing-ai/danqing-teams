package model

import "time"

type TaskStatus string

const (
	TaskPending          TaskStatus = "pending"
	TaskDispatching      TaskStatus = "dispatching"
	TaskRunning          TaskStatus = "running"
	TaskAwaitingApproval TaskStatus = "awaiting_approval"
	TaskReviewing        TaskStatus = "reviewing"
	TaskCompleted        TaskStatus = "completed"
	TaskFailed           TaskStatus = "failed"
)

type TaskCloseReason string

const (
	CloseReasonNone      TaskCloseReason = ""
	CloseReasonDone      TaskCloseReason = "done"
	CloseReasonNoIntent  TaskCloseReason = "no_intent"
	CloseReasonExhausted TaskCloseReason = "exhausted"
	CloseReasonCancelled TaskCloseReason = "cancelled"
	CloseReasonError     TaskCloseReason = "error"
)

type RunStatus string

const (
	RunQueued            RunStatus = "queued"
	RunPlanning          RunStatus = "planning"
	RunAwaitingApproval  RunStatus = "awaiting_approval"
	RunRunning           RunStatus = "running"
	RunCompleted         RunStatus = "completed"
	RunFailed            RunStatus = "failed"
	RunRejected          RunStatus = "rejected"
)

type ReportIntent string

const (
	ReportFinal          ReportIntent = "final"
	ReportNeedsFollowUp  ReportIntent = "needs_follow_up"
	ReportBlocked        ReportIntent = "blocked"
)

type Dispatch struct {
	ID             string
	TaskID         string
	WorkerID       string
	WorkerName     string
	Intent         string
	ContextSummary string
	Round          int
	CreatedAt      time.Time
}

type ExecutionPlan struct {
	RunID         string
	SkillIDs      []string
	ToolIDs       []string
	Rationale     string
	EvaluatedRisk RiskLevel
	HighRiskItems []RiskItem
}

type RiskItem struct {
	Type        string // skill | mcp_tool
	ID          string
	DisplayName string
}

type WorkerRun struct {
	ID         string
	TaskID     string
	DispatchID string
	WorkerID   string
	Status     RunStatus
	Plan       *ExecutionPlan
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Report struct {
	ID               string
	RunID            string
	TaskID           string
	WorkerID         string
	WorkerName       string
	ContentMarkdown  string
	Intent           ReportIntent
	SuggestedActions []SuggestedAction
	CreatedAt        time.Time
}

type SuggestedAction struct {
	Description       string
	TargetPersonaHint string
}

type TeamTask struct {
	ID          string
	TeamID      string
	Content     string
	Status      TaskStatus
	CloseReason TaskCloseReason
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TimelineEvent struct {
	ID        string
	TaskID    string
	Type      string
	Payload   any
	CreatedAt time.Time
}

type SubmitTaskRequest struct {
	Content     string
	Attachments []string
}

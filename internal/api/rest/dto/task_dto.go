package dto

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
	ID             string `json:"id"`
	TaskID         string `json:"taskId"`
	WorkerID       string `json:"workerId"`
	WorkerName     string `json:"workerName,omitempty"`
	Intent         string `json:"intent"`
	ContextSummary string `json:"contextSummary"`
	Round          int    `json:"round"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ExecutionPlan struct {
	RunID         string     `json:"runId"`
	SkillIDs      []string   `json:"skillIds"`
	ToolIDs       []string   `json:"toolIds"`
	Rationale     string     `json:"rationale"`
	EvaluatedRisk RiskLevel  `json:"evaluatedRisk"`
	HighRiskItems []RiskItem `json:"highRiskItems,omitempty"`
}

type RiskItem struct {
	Type        string `json:"type"` // skill | mcp_tool
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type WorkerRun struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"taskId"`
	DispatchID string     `json:"dispatchId"`
	WorkerID   string     `json:"workerId"`
	Status     RunStatus  `json:"status"`
	Plan       *ExecutionPlan `json:"plan,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type Report struct {
	ID               string           `json:"id"`
	RunID            string           `json:"runId"`
	TaskID           string           `json:"taskId"`
	WorkerID         string           `json:"workerId"`
	WorkerName       string           `json:"workerName,omitempty"`
	ContentMarkdown  string           `json:"contentMarkdown"`
	Intent           ReportIntent     `json:"intent"`
	SuggestedActions []SuggestedAction `json:"suggestedActions,omitempty"`
	CreatedAt        time.Time        `json:"createdAt"`
}

type SuggestedAction struct {
	Description       string `json:"description"`
	TargetPersonaHint string `json:"targetPersonaHint"`
}

type TeamTask struct {
	ID          string          `json:"id"`
	TeamID      string          `json:"teamId"`
	Content     string          `json:"content"`
	Status      TaskStatus      `json:"status"`
	CloseReason TaskCloseReason `json:"closeReason,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type TimelineEvent struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"taskId"`
	Type      string    `json:"type"`
	Payload   any       `json:"payload"`
	CreatedAt time.Time `json:"createdAt"`
}

type SubmitTaskRequest struct {
	Content     string   `json:"content"`
	Attachments []string `json:"attachments,omitempty"`
}

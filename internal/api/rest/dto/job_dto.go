package dto

import (
	"context"
	"time"
)

type JobKind string

const (
	JobRunTask    JobKind = "run_task"
	JobResumeRun  JobKind = "resume_run"
)

type JobStatus string

const (
	JobPending    JobStatus = "pending"
	JobProcessing JobStatus = "processing"
	JobCompleted  JobStatus = "completed"
	JobFailed     JobStatus = "failed"
)

// RunTaskPayload is serialized into OrchestrationJob.Payload for JobRunTask.
type RunTaskPayload struct {
	Intent         string `json:"intent"`
	Round          int    `json:"round"`
	ContextSummary string `json:"contextSummary,omitempty"`
}

// ResumeRunPayload is serialized into OrchestrationJob.Payload for JobResumeRun.
type ResumeRunPayload struct {
	RunID string `json:"runId"`
}

type OrchestrationJob struct {
	ID         string    `json:"id"`
	TeamID     string    `json:"teamId"`
	TaskID     string    `json:"taskId"`
	Kind       JobKind   `json:"kind"`
	Payload    string    `json:"payload"` // JSON
	DedupKey   string    `json:"dedupKey"`
	Status     JobStatus `json:"status"`
	LeaseOwner string    `json:"leaseOwner,omitempty"`
	LeaseUntil time.Time `json:"leaseUntil,omitempty"`
	LastError  string    `json:"lastError,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type JobRepository interface {
	Enqueue(ctx context.Context, job *OrchestrationJob) error
	ClaimNext(ctx context.Context, instanceID string, leaseUntil time.Time) (*OrchestrationJob, error)
	Complete(ctx context.Context, jobID string) error
	Fail(ctx context.Context, jobID string, errMsg string) error
	ReleaseExpiredLeases(ctx context.Context) (int, error)
	HasActiveJobForTask(ctx context.Context, taskID string) (bool, error)
}

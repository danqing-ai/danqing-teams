package model

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
	Intent         string
	Round          int
	ContextSummary string
}

// ResumeRunPayload is serialized into OrchestrationJob.Payload for JobResumeRun.
type ResumeRunPayload struct {
	RunID string
}

type OrchestrationJob struct {
	ID         string
	TeamID     string
	TaskID     string
	Kind       JobKind
	Payload    string // JSON
	DedupKey   string
	Status     JobStatus
	LeaseOwner string
	LeaseUntil time.Time
	LastError  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type JobRepository interface {
	Enqueue(ctx context.Context, job *OrchestrationJob) error
	ClaimNext(ctx context.Context, instanceID string, leaseUntil time.Time) (*OrchestrationJob, error)
	Complete(ctx context.Context, jobID string) error
	Fail(ctx context.Context, jobID string, errMsg string) error
	ReleaseExpiredLeases(ctx context.Context) (int, error)
	HasActiveJobForTask(ctx context.Context, taskID string) (bool, error)
}

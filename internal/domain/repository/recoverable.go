package repository

import (
	"context"

	"danqing-teams/internal/domain/model"
)

// RecoverableTaskStore lists tasks that may need orchestration jobs after restart.
type RecoverableTaskStore interface {
	ListRecoverableTasks(ctx context.Context) ([]model.TeamTask, error)
	LastUserMessage(ctx context.Context, taskID string) (string, error)
	GetDispatchByRound(ctx context.Context, taskID string, round int) (*model.Dispatch, error)
	GetRunByDispatchID(ctx context.Context, dispatchID string) (*model.WorkerRun, error)
	GetReportByRunID(ctx context.Context, runID string) (*model.Report, error)
}

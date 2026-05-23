package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/id"
)

func runTaskDedupKey(taskID string, round int) string {
	return fmt.Sprintf("run_task:%s:%d", taskID, round)
}

func resumeRunDedupKey(runID string) string {
	return "resume_run:" + runID
}

func (s *OrchestrationService) enqueueRunTask(ctx context.Context, teamID, taskID, intent string, round int, contextSummary string) error {
	if s.jobs == nil {
		go s.runTask(context.Background(), teamID, &contract.TeamTask{ID: taskID, TeamID: teamID}, intent, round, contextSummary)
		return nil
	}
	payload, _ := json.Marshal(contract.RunTaskPayload{
		Intent: intent, Round: round, ContextSummary: contextSummary,
	})
	now := nowUTC()
	return s.jobs.Enqueue(ctx, &contract.OrchestrationJob{
		ID: id.New(), TeamID: teamID, TaskID: taskID,
		Kind: contract.JobRunTask, Payload: string(payload),
		DedupKey: runTaskDedupKey(taskID, round),
		Status: contract.JobPending, CreatedAt: now, UpdatedAt: now,
	})
}

func (s *OrchestrationService) enqueueResumeRun(ctx context.Context, teamID, taskID, runID string) error {
	if s.jobs == nil {
		go func() { _ = s.ResumeRunAfterApproval(context.Background(), teamID, runID) }()
		return nil
	}
	payload, _ := json.Marshal(contract.ResumeRunPayload{RunID: runID})
	now := nowUTC()
	return s.jobs.Enqueue(ctx, &contract.OrchestrationJob{
		ID: id.New(), TeamID: teamID, TaskID: taskID,
		Kind: contract.JobResumeRun, Payload: string(payload),
		DedupKey: resumeRunDedupKey(runID),
		Status: contract.JobPending, CreatedAt: now, UpdatedAt: now,
	})
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

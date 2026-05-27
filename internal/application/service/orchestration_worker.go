package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/domain/repository"
	"danqing-teams/pkg/errs"
)

const (
	defaultLeaseDuration = 2 * time.Minute
	defaultPollInterval  = 500 * time.Millisecond
)

type OrchestrationWorker struct {
	orch           *OrchestrationService
	jobs           repository.JobRepository
	recover        repository.RecoverableTaskStore
	instanceID     string
	leaseDuration  time.Duration
	pollInterval   time.Duration
}

func NewOrchestrationWorker(
	orch *OrchestrationService,
	jobs repository.JobRepository,
	recover repository.RecoverableTaskStore,
	instanceID string,
) *OrchestrationWorker {
	if instanceID == "" {
		instanceID = "instance-" + idNewShort()
	}
	return &OrchestrationWorker{
		orch: orch, jobs: jobs, recover: recover, instanceID: instanceID,
		leaseDuration: defaultLeaseDuration, pollInterval: defaultPollInterval,
	}
}

func (w *OrchestrationWorker) Start(ctx context.Context) {
	if w.jobs == nil {
		return
	}
	if err := w.Recover(ctx); err != nil {
		log.Printf("orchestration recover: %v", err)
	}
	go w.loop(ctx)
}

func (w *OrchestrationWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.pollOnce(ctx)
		}
	}
}

func (w *OrchestrationWorker) pollOnce(ctx context.Context) {
	leaseUntil := time.Now().UTC().Add(w.leaseDuration)
	job, err := w.jobs.ClaimNext(ctx, w.instanceID, leaseUntil)
	if err != nil || job == nil {
		return
	}
	if err := w.processJob(ctx, job); err != nil {
		_ = w.jobs.Fail(ctx, job.ID, err.Error())
		return
	}
	_ = w.jobs.Complete(ctx, job.ID)
}

func (w *OrchestrationWorker) processJob(ctx context.Context, job *model.OrchestrationJob) error {
	switch job.Kind {
	case model.JobRunTask:
		var p model.RunTaskPayload
		if err := json.Unmarshal([]byte(job.Payload), &p); err != nil {
			return err
		}
		task, err := w.orch.tasks.GetTask(ctx, job.TeamID, job.TaskID)
		if err != nil {
			return err
		}
		w.orch.runTask(ctx, job.TeamID, task, p.Intent, p.Round, p.ContextSummary)
		task, err = w.orch.tasks.GetTask(ctx, job.TeamID, job.TaskID)
		if err != nil {
			return err
		}
		if task.Status == model.TaskFailed {
			return errs.BadRequest("orchestration failed")
		}
		return nil
	case model.JobResumeRun:
		var p model.ResumeRunPayload
		if err := json.Unmarshal([]byte(job.Payload), &p); err != nil {
			return err
		}
		return w.orch.ResumeRunAfterApproval(ctx, job.TeamID, p.RunID)
	default:
		return errs.BadRequest("unknown job kind")
	}
}

func (w *OrchestrationWorker) Recover(ctx context.Context) error {
	if w.recover == nil {
		return nil
	}
	if n, err := w.jobs.ReleaseExpiredLeases(ctx); err != nil {
		return err
	} else if n > 0 {
		log.Printf("orchestration: released %d expired job leases", n)
	}

	tasks, err := w.recover.ListRecoverableTasks(ctx)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		active, err := w.jobs.HasActiveJobForTask(ctx, task.ID)
		if err != nil || active {
			continue
		}
		switch task.Status {
		case model.TaskDispatching:
			intent, err := w.recover.LastUserMessage(ctx, task.ID)
			if err != nil {
				continue
			}
			dispatches, _ := w.orch.tasks.ListDispatches(ctx, task.ID)
			if err := w.orch.enqueueRunTask(ctx, task.TeamID, task.ID, intent, len(dispatches), ""); err != nil {
				log.Printf("recover enqueue run_task task=%s: %v", task.ID, err)
			}
		case model.TaskRunning:
			if err := w.recoverRunningTask(ctx, task); err != nil {
				log.Printf("recover running task=%s: %v", task.ID, err)
			}
		}
	}
	return nil
}

func (w *OrchestrationWorker) recoverRunningTask(ctx context.Context, task model.TeamTask) error {
	runs, err := w.orch.tasks.ListRuns(ctx, task.ID)
	if err != nil || len(runs) == 0 {
		intent, err := w.recover.LastUserMessage(ctx, task.ID)
		if err != nil {
			return err
		}
		dispatches, _ := w.orch.tasks.ListDispatches(ctx, task.ID)
		return w.orch.enqueueRunTask(ctx, task.TeamID, task.ID, intent, len(dispatches), "")
	}
	latest := runs[len(runs)-1]
	switch latest.Status {
	case model.RunAwaitingApproval:
		return nil
	case model.RunPlanning, model.RunRunning:
		if _, err := w.recover.GetReportByRunID(ctx, latest.ID); err == nil {
			return nil
		}
		return w.orch.enqueueResumeRun(ctx, task.TeamID, task.ID, latest.ID)
	default:
		return nil
	}
}

func idNewShort() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

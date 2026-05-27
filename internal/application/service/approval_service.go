package service

import (
	"context"
	"time"

	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/domain/repository"
	"danqing-teams/pkg/errs"
)

type ApprovalService struct {
	teams    repository.TeamRepository
	tasks    repository.TaskRepository
	approvals repository.ApprovalRepository
	events   model.EventPublisher
	orch     *OrchestrationService
}

func NewApprovalService(
	teams repository.TeamRepository,
	tasks repository.TaskRepository,
	approvals repository.ApprovalRepository,
	events model.EventPublisher,
	orch *OrchestrationService,
) *ApprovalService {
	return &ApprovalService{teams: teams, tasks: tasks, approvals: approvals, events: events, orch: orch}
}

func (s *ApprovalService) ListPending(ctx context.Context, teamID string) ([]model.ApprovalRequest, error) {
	return s.approvals.List(ctx, teamID, model.ApprovalPending)
}

func (s *ApprovalService) Get(ctx context.Context, teamID, approvalID string) (*model.ApprovalRequest, error) {
	return s.approvals.Get(ctx, teamID, approvalID)
}

func (s *ApprovalService) Approve(ctx context.Context, teamID, approvalID string, req model.DecideApprovalRequest) (*model.ApprovalRequest, error) {
	return s.decide(ctx, teamID, approvalID, model.ApprovalApproved, req.Comment)
}

func (s *ApprovalService) Reject(ctx context.Context, teamID, approvalID string, req model.DecideApprovalRequest) (*model.ApprovalRequest, error) {
	return s.decide(ctx, teamID, approvalID, model.ApprovalRejected, req.Comment)
}

func (s *ApprovalService) decide(ctx context.Context, teamID, approvalID string, status model.ApprovalStatus, comment string) (*model.ApprovalRequest, error) {
	a, err := s.approvals.Get(ctx, teamID, approvalID)
	if err != nil {
		return nil, err
	}
	if a.Status != model.ApprovalPending {
		return nil, errs.BadRequest("approval already decided")
	}
	now := time.Now().UTC()
	a.Status = status
	a.Comment = comment
	a.UpdatedAt = now
	if err := s.approvals.Update(ctx, a); err != nil {
		return nil, err
	}

	run, err := s.tasks.GetRun(ctx, a.RunID)
	if err != nil {
		return nil, err
	}
	if status == model.ApprovalApproved {
		run.Status = model.RunRunning
		_ = s.tasks.UpdateRun(ctx, run)
		s.events.Publish(ctx, teamID, a.TaskID, model.StreamEvent{
			Type: model.EventApprovalApproved, Payload: a,
		})
		_ = s.orch.enqueueResumeRun(ctx, teamID, a.TaskID, run.ID)
	} else {
		run.Status = model.RunRejected
		_ = s.tasks.UpdateRun(ctx, run)
		_ = s.tasks.UpdateTaskClosure(ctx, teamID, a.TaskID, model.TaskFailed, model.CloseReasonError)
		s.events.Publish(ctx, teamID, a.TaskID, model.StreamEvent{
			Type: model.EventApprovalRejected, Payload: a,
		})
	}
	return a, nil
}

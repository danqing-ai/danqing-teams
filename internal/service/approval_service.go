package service

import (
	"context"
	"time"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
)

type ApprovalService struct {
	teams    contract.TeamRepository
	tasks    contract.TaskRepository
	approvals contract.ApprovalRepository
	events   contract.EventPublisher
	orch     *OrchestrationService
}

func NewApprovalService(
	teams contract.TeamRepository,
	tasks contract.TaskRepository,
	approvals contract.ApprovalRepository,
	events contract.EventPublisher,
	orch *OrchestrationService,
) *ApprovalService {
	return &ApprovalService{teams: teams, tasks: tasks, approvals: approvals, events: events, orch: orch}
}

func (s *ApprovalService) ListPending(ctx context.Context, teamID string) ([]contract.ApprovalRequest, error) {
	return s.approvals.List(ctx, teamID, contract.ApprovalPending)
}

func (s *ApprovalService) Get(ctx context.Context, teamID, approvalID string) (*contract.ApprovalRequest, error) {
	return s.approvals.Get(ctx, teamID, approvalID)
}

func (s *ApprovalService) Approve(ctx context.Context, teamID, approvalID string, req contract.DecideApprovalRequest) (*contract.ApprovalRequest, error) {
	return s.decide(ctx, teamID, approvalID, contract.ApprovalApproved, req.Comment)
}

func (s *ApprovalService) Reject(ctx context.Context, teamID, approvalID string, req contract.DecideApprovalRequest) (*contract.ApprovalRequest, error) {
	return s.decide(ctx, teamID, approvalID, contract.ApprovalRejected, req.Comment)
}

func (s *ApprovalService) decide(ctx context.Context, teamID, approvalID string, status contract.ApprovalStatus, comment string) (*contract.ApprovalRequest, error) {
	a, err := s.approvals.Get(ctx, teamID, approvalID)
	if err != nil {
		return nil, err
	}
	if a.Status != contract.ApprovalPending {
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
	if status == contract.ApprovalApproved {
		run.Status = contract.RunRunning
		_ = s.tasks.UpdateRun(ctx, run)
		s.events.Publish(ctx, teamID, a.TaskID, contract.StreamEvent{
			Type: contract.EventApprovalApproved, Payload: a,
		})
		_ = s.orch.enqueueResumeRun(ctx, teamID, a.TaskID, run.ID)
	} else {
		run.Status = contract.RunRejected
		_ = s.tasks.UpdateRun(ctx, run)
		_ = s.tasks.UpdateTaskClosure(ctx, teamID, a.TaskID, contract.TaskFailed, contract.CloseReasonError)
		s.events.Publish(ctx, teamID, a.TaskID, contract.StreamEvent{
			Type: contract.EventApprovalRejected, Payload: a,
		})
	}
	return a, nil
}

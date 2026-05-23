package service

import (
	"context"

	"danqing-teams/internal/contract"
)

type TaskService struct {
	tasks         contract.TaskRepository
	orchestration *OrchestrationService
}

func NewTaskService(tasks contract.TaskRepository, orch *OrchestrationService) *TaskService {
	return &TaskService{tasks: tasks, orchestration: orch}
}

func (s *TaskService) List(ctx context.Context, teamID string, status contract.TaskStatus) ([]contract.TeamTask, error) {
	return s.tasks.ListTasks(ctx, teamID, status)
}

func (s *TaskService) Get(ctx context.Context, teamID, taskID string) (*contract.TeamTask, error) {
	return s.tasks.GetTask(ctx, teamID, taskID)
}

func (s *TaskService) Submit(ctx context.Context, teamID string, req contract.SubmitTaskRequest) (*contract.TeamTask, error) {
	return s.orchestration.SubmitTask(ctx, teamID, req)
}

func (s *TaskService) SendMessage(ctx context.Context, teamID string, req contract.SendTeamMessageRequest) (*contract.SendTeamMessageResponse, error) {
	return s.orchestration.SendTeamMessage(ctx, teamID, req)
}

func (s *TaskService) ListMessages(ctx context.Context, teamID, taskID string) ([]contract.TeamMessage, error) {
	return s.orchestration.ListMessages(ctx, teamID, taskID)
}

func (s *TaskService) Timeline(ctx context.Context, teamID, taskID string) ([]contract.TimelineEvent, error) {
	return s.orchestration.GetTimeline(ctx, teamID, taskID)
}

func (s *TaskService) Reports(ctx context.Context, _, taskID string) ([]contract.Report, error) {
	return s.tasks.ListReports(ctx, taskID)
}

func (s *TaskService) GetPlan(ctx context.Context, teamID, taskID, runID string) (*contract.ExecutionPlan, error) {
	return s.orchestration.GetPlan(ctx, teamID, taskID, runID)
}

func (s *TaskService) Cancel(ctx context.Context, teamID, taskID string) error {
	return s.orchestration.CancelTask(ctx, teamID, taskID)
}

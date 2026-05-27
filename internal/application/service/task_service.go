package service

import (
	"context"

	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/domain/repository"
)

type TaskService struct {
	tasks         repository.TaskRepository
	orchestration *OrchestrationService
}

func NewTaskService(tasks repository.TaskRepository, orch *OrchestrationService) *TaskService {
	return &TaskService{tasks: tasks, orchestration: orch}
}

func (s *TaskService) List(ctx context.Context, teamID string, status model.TaskStatus) ([]model.TeamTask, error) {
	return s.tasks.ListTasks(ctx, teamID, status)
}

func (s *TaskService) Get(ctx context.Context, teamID, taskID string) (*model.TeamTask, error) {
	return s.tasks.GetTask(ctx, teamID, taskID)
}

func (s *TaskService) Submit(ctx context.Context, teamID string, req model.SubmitTaskRequest) (*model.TeamTask, error) {
	return s.orchestration.SubmitTask(ctx, teamID, req)
}

func (s *TaskService) SendMessage(ctx context.Context, teamID string, req model.SendTeamMessageRequest) (*model.SendTeamMessageResponse, error) {
	return s.orchestration.SendTeamMessage(ctx, teamID, req)
}

func (s *TaskService) ListMessages(ctx context.Context, teamID, taskID string) ([]model.TeamMessage, error) {
	return s.orchestration.ListMessages(ctx, teamID, taskID)
}

func (s *TaskService) Timeline(ctx context.Context, teamID, taskID string) ([]model.TimelineEvent, error) {
	return s.orchestration.GetTimeline(ctx, teamID, taskID)
}

func (s *TaskService) Reports(ctx context.Context, _, taskID string) ([]model.Report, error) {
	return s.tasks.ListReports(ctx, taskID)
}

func (s *TaskService) GetPlan(ctx context.Context, teamID, taskID, runID string) (*model.ExecutionPlan, error) {
	return s.orchestration.GetPlan(ctx, teamID, taskID, runID)
}

func (s *TaskService) Cancel(ctx context.Context, teamID, taskID string) error {
	return s.orchestration.CancelTask(ctx, teamID, taskID)
}

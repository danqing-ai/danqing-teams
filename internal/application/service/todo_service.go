package service

import (
	"context"

	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/domain/repository"
	"danqing-teams/pkg/id"
)

type TodoService struct {
	store repository.TeamRepository
}

func NewTodoService(store repository.TeamRepository) *TodoService {
	return &TodoService{store: store}
}

func (s *TodoService) List(ctx context.Context, teamID, taskID string) ([]model.TodoItem, error) {
	return s.store.ListTodos(ctx, teamID, taskID)
}

func (s *TodoService) Create(ctx context.Context, teamID, title, taskID string) (*model.TodoItem, error) {
	return s.store.CreateTodo(ctx, teamID, model.TodoItem{
		ID: id.New(), Title: title, TaskID: taskID,
	})
}

func (s *TodoService) Update(ctx context.Context, teamID, todoID string, done bool) (*model.TodoItem, error) {
	return s.store.UpdateTodo(ctx, teamID, todoID, done)
}

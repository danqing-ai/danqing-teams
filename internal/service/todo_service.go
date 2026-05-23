package service

import (
	"context"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/id"
)

type TodoService struct {
	store contract.TeamRepository
}

func NewTodoService(store contract.TeamRepository) *TodoService {
	return &TodoService{store: store}
}

func (s *TodoService) List(ctx context.Context, teamID, taskID string) ([]contract.TodoItem, error) {
	return s.store.ListTodos(ctx, teamID, taskID)
}

func (s *TodoService) Create(ctx context.Context, teamID, title, taskID string) (*contract.TodoItem, error) {
	return s.store.CreateTodo(ctx, teamID, contract.TodoItem{
		ID: id.New(), Title: title, TaskID: taskID,
	})
}

func (s *TodoService) Update(ctx context.Context, teamID, todoID string, done bool) (*contract.TodoItem, error) {
	return s.store.UpdateTodo(ctx, teamID, todoID, done)
}

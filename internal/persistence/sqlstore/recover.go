package sqlstore

import (
	"context"

	"danqing-teams/internal/domain/model"
)

func (s *Store) ListRecoverableTasks(ctx context.Context) ([]model.TeamTask, error) {
	return s.ListTasksByStatuses(ctx, model.TaskDispatching, model.TaskRunning)
}

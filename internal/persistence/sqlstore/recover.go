package sqlstore

import (
	"context"

	"danqing-teams/internal/contract"
)

func (s *Store) ListRecoverableTasks(ctx context.Context) ([]contract.TeamTask, error) {
	return s.ListTasksByStatuses(ctx, contract.TaskDispatching, contract.TaskRunning)
}

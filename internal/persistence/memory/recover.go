package memory

import (
	"context"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
)

func (s *Store) ListRecoverableTasks(ctx context.Context) ([]contract.TeamTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []contract.TeamTask
	for _, t := range s.tasks {
		if t.Status == contract.TaskDispatching || t.Status == contract.TaskRunning {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (s *Store) LastUserMessage(_ context.Context, taskID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[taskID]
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == contract.MessageRoleUser {
			return msgs[i].Content, nil
		}
	}
	return "", errs.NotFound("message not found")
}

func (s *Store) GetDispatchByRound(_ context.Context, taskID string, round int) (*contract.Dispatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.dispatches[taskID] {
		if d.Round == round {
			cp := d
			return &cp, nil
		}
	}
	return nil, errs.NotFound("dispatch not found")
}

func (s *Store) GetRunByDispatchID(_ context.Context, dispatchID string) (*contract.WorkerRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.runs {
		if r.DispatchID == dispatchID {
			cp := *r
			return &cp, nil
		}
	}
	return nil, errs.NotFound("run not found")
}

func (s *Store) GetReportByRunID(_ context.Context, runID string) (*contract.Report, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, reps := range s.reports {
		for _, r := range reps {
			if r.RunID == runID {
				cp := r
				return &cp, nil
			}
		}
	}
	return nil, errs.NotFound("report not found")
}

package memory

import (
	"context"
	"sort"
	"time"

	"danqing-teams/internal/domain/model"
	"danqing-teams/pkg/errs"
)

func (s *Store) Enqueue(_ context.Context, job *model.OrchestrationJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.jobs == nil {
		s.jobs = make(map[string]*model.OrchestrationJob)
	}
	for _, j := range s.jobs {
		if j.DedupKey == job.DedupKey && (j.Status == model.JobPending || j.Status == model.JobProcessing) {
			return nil
		}
	}
	s.jobs[job.ID] = job
	return nil
}

func (s *Store) ClaimNext(_ context.Context, instanceID string, leaseUntil time.Time) (*model.OrchestrationJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	var candidates []*model.OrchestrationJob
	for _, j := range s.jobs {
		if j.Status == model.JobPending {
			candidates = append(candidates, j)
			continue
		}
		if j.Status == model.JobProcessing && !j.LeaseUntil.IsZero() && j.LeaseUntil.Before(now) {
			candidates = append(candidates, j)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].CreatedAt.Before(candidates[j].CreatedAt)
	})
	job := candidates[0]
	job.Status = model.JobProcessing
	job.LeaseOwner = instanceID
	job.LeaseUntil = leaseUntil
	job.UpdatedAt = now
	return job, nil
}

func (s *Store) Complete(_ context.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[jobID]
	if !ok {
		return errs.NotFound("job not found")
	}
	j.Status = model.JobCompleted
	j.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) Fail(_ context.Context, jobID string, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[jobID]
	if !ok {
		return errs.NotFound("job not found")
	}
	j.Status = model.JobFailed
	j.LastError = errMsg
	j.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) ReleaseExpiredLeases(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	n := 0
	for _, j := range s.jobs {
		if j.Status == model.JobProcessing && !j.LeaseUntil.IsZero() && j.LeaseUntil.Before(now) {
			j.Status = model.JobPending
			j.LeaseOwner = ""
			j.LeaseUntil = time.Time{}
			j.UpdatedAt = now
			n++
		}
	}
	return n, nil
}

func (s *Store) HasActiveJobForTask(_ context.Context, taskID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, j := range s.jobs {
		if j.TaskID != taskID {
			continue
		}
		if j.Status == model.JobPending || j.Status == model.JobProcessing {
			return true, nil
		}
	}
	return false, nil
}

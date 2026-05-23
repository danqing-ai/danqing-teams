package sqlstore

import (
	"context"
	"errors"
	"time"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
	"gorm.io/gorm"
)

func (s *Store) Enqueue(ctx context.Context, job *contract.OrchestrationJob) error {
	var existing int64
	err := s.dbWithCtx(ctx).Model(&orchestrationJobRow{}).
		Where("dedup_key = ? AND status IN ?", job.DedupKey, []contract.JobStatus{contract.JobPending, contract.JobProcessing}).
		Count(&existing).Error
	if err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}
	payload := job.Payload
	if payload == "" {
		payload = "{}"
	}
	return s.dbWithCtx(ctx).Create(&orchestrationJobRow{
		ID: job.ID, TeamID: job.TeamID, TaskID: job.TaskID, Kind: job.Kind,
		Payload: payload, DedupKey: job.DedupKey, Status: job.Status,
		CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
	}).Error
}

func (s *Store) ClaimNext(ctx context.Context, instanceID string, leaseUntil time.Time) (*contract.OrchestrationJob, error) {
	var claimed *contract.OrchestrationJob
	now := nowUTC()
	err := s.dbWithCtx(ctx).Transaction(func(tx *gorm.DB) error {
		var row orchestrationJobRow
		err := tx.Where(
			"status = ? OR (status = ? AND lease_until IS NOT NULL AND lease_until < ?)",
			contract.JobPending, contract.JobProcessing, now,
		).Order("created_at").First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		res := tx.Model(&orchestrationJobRow{}).Where(
			"id = ? AND (status = ? OR (status = ? AND lease_until IS NOT NULL AND lease_until < ?))",
			row.ID, contract.JobPending, contract.JobProcessing, now,
		).Updates(map[string]any{
			"status": contract.JobProcessing, "lease_owner": instanceID,
			"lease_until": leaseUntil, "updated_at": now,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		if err := tx.First(&row, "id = ?", row.ID).Error; err != nil {
			return err
		}
		j := jobFromRow(row)
		claimed = &j
		return nil
	})
	if err != nil || claimed == nil {
		return nil, err
	}
	return claimed, nil
}

func (s *Store) Complete(ctx context.Context, jobID string) error {
	res := s.dbWithCtx(ctx).Model(&orchestrationJobRow{}).Where("id = ?", jobID).Updates(map[string]any{
		"status": contract.JobCompleted, "updated_at": nowUTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("job not found")
	}
	return nil
}

func (s *Store) Fail(ctx context.Context, jobID string, errMsg string) error {
	res := s.dbWithCtx(ctx).Model(&orchestrationJobRow{}).Where("id = ?", jobID).Updates(map[string]any{
		"status": contract.JobFailed, "last_error": errMsg, "updated_at": nowUTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("job not found")
	}
	return nil
}

func (s *Store) ReleaseExpiredLeases(ctx context.Context) (int, error) {
	now := nowUTC()
	res := s.dbWithCtx(ctx).Model(&orchestrationJobRow{}).Where(
		"status = ? AND lease_until IS NOT NULL AND lease_until < ?",
		contract.JobProcessing, now,
	).Updates(map[string]any{
		"status": contract.JobPending, "lease_owner": "", "lease_until": nil, "updated_at": now,
	})
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

func (s *Store) HasActiveJobForTask(ctx context.Context, taskID string) (bool, error) {
	var n int64
	err := s.dbWithCtx(ctx).Model(&orchestrationJobRow{}).
		Where("task_id = ? AND status IN ?", taskID, []contract.JobStatus{contract.JobPending, contract.JobProcessing}).
		Count(&n).Error
	return n > 0, err
}

package sqlstore

import (
	"context"
	"database/sql"
	"time"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
)

func (s *Store) Enqueue(ctx context.Context, job *contract.OrchestrationJob) error {
	var existing int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM orchestration_jobs
		 WHERE dedup_key = ? AND status IN (?, ?)`,
		job.DedupKey, string(contract.JobPending), string(contract.JobProcessing),
	).Scan(&existing)
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
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO orchestration_jobs
		 (id, team_id, task_id, kind, payload_json, dedup_key, status, lease_owner, lease_until, last_error, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, '', NULL, '', ?, ?)`,
		job.ID, job.TeamID, job.TaskID, string(job.Kind), payload, job.DedupKey, string(job.Status),
		formatTime(job.CreatedAt), formatTime(job.UpdatedAt),
	)
	return err
}

func (s *Store) ClaimNext(ctx context.Context, instanceID string, leaseUntil time.Time) (*contract.OrchestrationJob, error) {
	var job *contract.OrchestrationJob
	now := nowUTC()
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx,
			`SELECT id FROM orchestration_jobs
			 WHERE status = ? OR (status = ? AND lease_until IS NOT NULL AND lease_until < ?)
			 ORDER BY created_at LIMIT 1`,
			string(contract.JobPending), string(contract.JobProcessing), formatTime(now),
		)
		var id string
		if err := row.Scan(&id); err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		res, err := tx.ExecContext(ctx,
			`UPDATE orchestration_jobs
			 SET status = ?, lease_owner = ?, lease_until = ?, updated_at = ?
			 WHERE id = ? AND (status = ? OR (status = ? AND lease_until IS NOT NULL AND lease_until < ?))`,
			string(contract.JobProcessing), instanceID, formatTime(leaseUntil), formatTime(now), id,
			string(contract.JobPending), string(contract.JobProcessing), formatTime(now),
		)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return nil
		}
		job, err = scanJob(tx.QueryRowContext(ctx,
			`SELECT id, team_id, task_id, kind, payload_json, dedup_key, status, lease_owner, lease_until, last_error, created_at, updated_at
			 FROM orchestration_jobs WHERE id = ?`, id,
		))
		return err
	})
	if err != nil || job == nil {
		return nil, err
	}
	return job, nil
}

func (s *Store) Complete(ctx context.Context, jobID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orchestration_jobs SET status = ?, updated_at = ? WHERE id = ?`,
		string(contract.JobCompleted), formatTime(nowUTC()), jobID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errs.NotFound("job not found")
	}
	return nil
}

func (s *Store) Fail(ctx context.Context, jobID string, errMsg string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE orchestration_jobs SET status = ?, last_error = ?, updated_at = ? WHERE id = ?`,
		string(contract.JobFailed), errMsg, formatTime(nowUTC()), jobID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errs.NotFound("job not found")
	}
	return nil
}

func (s *Store) ReleaseExpiredLeases(ctx context.Context) (int, error) {
	now := formatTime(nowUTC())
	res, err := s.db.ExecContext(ctx,
		`UPDATE orchestration_jobs
		 SET status = ?, lease_owner = '', lease_until = NULL, updated_at = ?
		 WHERE status = ? AND lease_until IS NOT NULL AND lease_until < ?`,
		string(contract.JobPending), now, string(contract.JobProcessing), now,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *Store) HasActiveJobForTask(ctx context.Context, taskID string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM orchestration_jobs
		 WHERE task_id = ? AND status IN (?, ?)`,
		taskID, string(contract.JobPending), string(contract.JobProcessing),
	).Scan(&n)
	return n > 0, err
}

func scanJob(row *sql.Row) (*contract.OrchestrationJob, error) {
	var j contract.OrchestrationJob
	var kind, status, payload, leaseOwner, leaseUntil, lastErr, created, updated sql.NullString
	if err := row.Scan(
		&j.ID, &j.TeamID, &j.TaskID, &kind, &payload, &j.DedupKey, &status,
		&leaseOwner, &leaseUntil, &lastErr, &created, &updated,
	); err != nil {
		return nil, err
	}
	j.Kind = contract.JobKind(kind.String)
	j.Payload = payload.String
	j.Status = contract.JobStatus(status.String)
	j.LeaseOwner = leaseOwner.String
	j.LastError = lastErr.String
	j.LeaseUntil, _ = scanOptionalTime(leaseUntil)
	j.CreatedAt, _ = parseTime(created.String)
	j.UpdatedAt, _ = parseTime(updated.String)
	return &j, nil
}

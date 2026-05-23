package sqlstore

import (
	"context"
	"database/sql"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
	"danqing-teams/pkg/id"
)

func (s *Store) Create(ctx context.Context, req *contract.ApprovalRequest) error {
	return s.createApproval(ctx, req)
}

func (s *Store) createApproval(ctx context.Context, req *contract.ApprovalRequest) error {
	high, _ := encodeJSON(req.HighRiskItems)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO approvals (id, team_id, task_id, run_id, summary, high_risk_json, status, comment, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID, req.TeamID, req.TaskID, req.RunID, req.Summary, high, string(req.Status), req.Comment,
		formatTime(req.CreatedAt), formatTime(req.UpdatedAt),
	)
	return err
}

func (s *Store) Get(ctx context.Context, teamID, approvalID string) (*contract.ApprovalRequest, error) {
	return s.GetApproval(ctx, teamID, approvalID)
}

func (s *Store) GetApproval(ctx context.Context, _, approvalID string) (*contract.ApprovalRequest, error) {
	a, err := scanApproval(s.db.QueryRowContext(ctx,
		`SELECT id, team_id, task_id, run_id, summary, high_risk_json, status, comment, created_at, updated_at
		 FROM approvals WHERE id = ?`, approvalID,
	))
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("approval not found")
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) Update(ctx context.Context, req *contract.ApprovalRequest) error {
	return s.UpdateApproval(ctx, req)
}

func (s *Store) UpdateApproval(ctx context.Context, req *contract.ApprovalRequest) error {
	high, _ := encodeJSON(req.HighRiskItems)
	res, err := s.db.ExecContext(ctx,
		`UPDATE approvals SET status = ?, comment = ?, high_risk_json = ?, summary = ?, updated_at = ? WHERE id = ?`,
		string(req.Status), req.Comment, high, req.Summary, formatTime(req.UpdatedAt), req.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errs.NotFound("approval not found")
	}
	return nil
}

func (s *Store) List(ctx context.Context, teamID string, status contract.ApprovalStatus) ([]contract.ApprovalRequest, error) {
	return s.ListApprovals(ctx, teamID, status)
}

func (s *Store) ListApprovals(ctx context.Context, teamID string, status contract.ApprovalStatus) ([]contract.ApprovalRequest, error) {
	q := `SELECT id, team_id, task_id, run_id, summary, high_risk_json, status, comment, created_at, updated_at FROM approvals WHERE team_id = ?`
	args := []any{teamID}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, string(status))
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.ApprovalRequest
	for rows.Next() {
		a, err := scanApproval(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetByRunID(ctx context.Context, runID string) (*contract.ApprovalRequest, error) {
	return s.GetApprovalByRunID(ctx, runID)
}

func (s *Store) GetApprovalByRunID(ctx context.Context, runID string) (*contract.ApprovalRequest, error) {
	a, err := scanApproval(s.db.QueryRowContext(ctx,
		`SELECT id, team_id, task_id, run_id, summary, high_risk_json, status, comment, created_at, updated_at
		 FROM approvals WHERE run_id = ?`, runID,
	))
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("approval not found")
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func scanApproval(scanner interface {
	Scan(dest ...any) error
}) (contract.ApprovalRequest, error) {
	var a contract.ApprovalRequest
	var high, status, created, updated string
	if err := scanner.Scan(
		&a.ID, &a.TeamID, &a.TaskID, &a.RunID, &a.Summary, &high, &status, &a.Comment, &created, &updated,
	); err != nil {
		return a, err
	}
	a.Status = contract.ApprovalStatus(status)
	a.CreatedAt, _ = parseTime(created)
	a.UpdatedAt, _ = parseTime(updated)
	_ = decodeJSON(high, &a.HighRiskItems)
	return a, nil
}

func (s *Store) ListTodos(ctx context.Context, teamID, taskID string) ([]contract.TodoItem, error) {
	q := `SELECT id, team_id, task_id, title, done, created_at FROM todos WHERE team_id = ?`
	args := []any{teamID}
	if taskID != "" {
		q += ` AND task_id = ?`
		args = append(args, taskID)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.TodoItem
	for rows.Next() {
		var it contract.TodoItem
		var done int
		var created string
		if err := rows.Scan(&it.ID, &it.TeamID, &it.TaskID, &it.Title, &done, &created); err != nil {
			return nil, err
		}
		it.Done = done != 0
		it.CreatedAt, _ = parseTime(created)
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) CreateTodo(ctx context.Context, teamID string, item contract.TodoItem) (*contract.TodoItem, error) {
	if item.ID == "" {
		item.ID = id.New()
	}
	item.TeamID = teamID
	item.CreatedAt = nowUTC()
	done := 0
	if item.Done {
		done = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO todos (id, team_id, task_id, title, done, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, teamID, item.TaskID, item.Title, done, formatTime(item.CreatedAt),
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) UpdateTodo(ctx context.Context, teamID, todoID string, done bool) (*contract.TodoItem, error) {
	doneInt := 0
	if done {
		doneInt = 1
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE todos SET done = ? WHERE team_id = ? AND id = ?`, doneInt, teamID, todoID,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, errs.NotFound("todo not found")
	}
	items, err := s.ListTodos(ctx, teamID, "")
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.ID == todoID {
			return &it, nil
		}
	}
	return nil, errs.NotFound("todo not found")
}

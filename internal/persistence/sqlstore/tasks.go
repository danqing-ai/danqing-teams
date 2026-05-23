package sqlstore

import (
	"context"
	"database/sql"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
)

func (s *Store) ListTasks(ctx context.Context, teamID string, status contract.TaskStatus) ([]contract.TeamTask, error) {
	q := `SELECT id, team_id, content, status, close_reason, created_at, updated_at FROM tasks WHERE team_id = ?`
	args := []any{teamID}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, string(status))
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (s *Store) GetTask(ctx context.Context, teamID, taskID string) (*contract.TeamTask, error) {
	var t contract.TeamTask
	var created, updated string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, team_id, content, status, close_reason, created_at, updated_at FROM tasks WHERE id = ?`, taskID,
	).Scan(&t.ID, &t.TeamID, &t.Content, &t.Status, &t.CloseReason, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("task not found")
	}
	if err != nil {
		return nil, err
	}
	if teamID != "" && t.TeamID != teamID {
		return nil, errs.NotFound("task not found")
	}
	t.CreatedAt, _ = parseTime(created)
	t.UpdatedAt, _ = parseTime(updated)
	return &t, nil
}

func (s *Store) CreateTask(ctx context.Context, task *contract.TeamTask) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (id, team_id, content, status, close_reason, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.TeamID, task.Content, string(task.Status), string(task.CloseReason),
		formatTime(task.CreatedAt), formatTime(task.UpdatedAt),
	)
	return err
}

func (s *Store) UpdateTaskStatus(ctx context.Context, _, taskID string, status contract.TaskStatus) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), formatTime(nowUTC()), taskID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errs.NotFound("task not found")
	}
	return nil
}

func (s *Store) UpdateTaskClosure(ctx context.Context, _, taskID string, status contract.TaskStatus, reason contract.TaskCloseReason) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, close_reason = ?, updated_at = ? WHERE id = ?`,
		string(status), string(reason), formatTime(nowUTC()), taskID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errs.NotFound("task not found")
	}
	return nil
}

func (s *Store) SaveDispatch(ctx context.Context, d *contract.Dispatch) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO dispatches (id, task_id, worker_id, worker_name, intent, context_summary, round_num, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.TaskID, d.WorkerID, d.WorkerName, d.Intent, d.ContextSummary, d.Round, formatTime(d.CreatedAt),
	)
	return err
}

func (s *Store) ListDispatches(ctx context.Context, taskID string) ([]contract.Dispatch, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, task_id, worker_id, worker_name, intent, context_summary, round_num, created_at
		 FROM dispatches WHERE task_id = ? ORDER BY created_at`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.Dispatch
	for rows.Next() {
		var d contract.Dispatch
		var created string
		if err := rows.Scan(&d.ID, &d.TaskID, &d.WorkerID, &d.WorkerName, &d.Intent, &d.ContextSummary, &d.Round, &created); err != nil {
			return nil, err
		}
		d.CreatedAt, _ = parseTime(created)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) SaveRun(ctx context.Context, run *contract.WorkerRun) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO worker_runs (id, task_id, dispatch_id, worker_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.TaskID, run.DispatchID, run.WorkerID, string(run.Status),
		formatTime(run.CreatedAt), formatTime(run.UpdatedAt),
	)
	return err
}

func (s *Store) GetRun(ctx context.Context, runID string) (*contract.WorkerRun, error) {
	run, err := scanRun(s.db.QueryRowContext(ctx,
		`SELECT id, task_id, dispatch_id, worker_id, status, created_at, updated_at FROM worker_runs WHERE id = ?`, runID,
	))
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("run not found")
	}
	if err != nil {
		return nil, err
	}
	plan, err := s.GetPlan(ctx, runID)
	if err == nil {
		run.Plan = plan
	}
	return run, nil
}

func (s *Store) UpdateRun(ctx context.Context, run *contract.WorkerRun) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE worker_runs SET status = ?, updated_at = ? WHERE id = ?`,
		string(run.Status), formatTime(nowUTC()), run.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errs.NotFound("run not found")
	}
	return nil
}

func (s *Store) ListRuns(ctx context.Context, taskID string) ([]contract.WorkerRun, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, task_id, dispatch_id, worker_id, status, created_at, updated_at
		 FROM worker_runs WHERE task_id = ? ORDER BY created_at`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.WorkerRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		plan, err := s.GetPlan(ctx, run.ID)
		if err == nil {
			run.Plan = plan
		}
		out = append(out, *run)
	}
	return out, rows.Err()
}

func (s *Store) SavePlan(ctx context.Context, plan *contract.ExecutionPlan) error {
	skills, _ := encodeJSON(plan.SkillIDs)
	tools, _ := encodeJSON(plan.ToolIDs)
	high, _ := encodeJSON(plan.HighRiskItems)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO execution_plans (run_id, skill_ids_json, tool_ids_json, rationale, evaluated_risk, high_risk_json)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(run_id) DO UPDATE SET skill_ids_json=excluded.skill_ids_json, tool_ids_json=excluded.tool_ids_json,
		   rationale=excluded.rationale, evaluated_risk=excluded.evaluated_risk, high_risk_json=excluded.high_risk_json`,
		plan.RunID, skills, tools, plan.Rationale, string(plan.EvaluatedRisk), high,
	)
	return err
}

func (s *Store) GetPlan(ctx context.Context, runID string) (*contract.ExecutionPlan, error) {
	var plan contract.ExecutionPlan
	var skills, tools, high, risk string
	err := s.db.QueryRowContext(ctx,
		`SELECT run_id, skill_ids_json, tool_ids_json, rationale, evaluated_risk, high_risk_json
		 FROM execution_plans WHERE run_id = ?`, runID,
	).Scan(&plan.RunID, &skills, &tools, &plan.Rationale, &risk, &high)
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("plan not found")
	}
	if err != nil {
		return nil, err
	}
	_ = decodeJSON(skills, &plan.SkillIDs)
	_ = decodeJSON(tools, &plan.ToolIDs)
	_ = decodeJSON(high, &plan.HighRiskItems)
	plan.EvaluatedRisk = contract.RiskLevel(risk)
	return &plan, nil
}

func (s *Store) SaveReport(ctx context.Context, r *contract.Report) error {
	actions, _ := encodeJSON(r.SuggestedActions)
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO reports (id, run_id, task_id, worker_id, worker_name, content_markdown, intent, suggested_actions_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.RunID, r.TaskID, r.WorkerID, r.WorkerName, r.ContentMarkdown, string(r.Intent), actions, formatTime(r.CreatedAt),
	)
	return err
}

func (s *Store) ListReports(ctx context.Context, taskID string) ([]contract.Report, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, run_id, task_id, worker_id, worker_name, content_markdown, intent, suggested_actions_json, created_at
		 FROM reports WHERE task_id = ? ORDER BY created_at`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.Report
	for rows.Next() {
		var r contract.Report
		var actions, created, intent string
		if err := rows.Scan(&r.ID, &r.RunID, &r.TaskID, &r.WorkerID, &r.WorkerName, &r.ContentMarkdown, &intent, &actions, &created); err != nil {
			return nil, err
		}
		r.Intent = contract.ReportIntent(intent)
		r.CreatedAt, _ = parseTime(created)
		_ = decodeJSON(actions, &r.SuggestedActions)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) AppendTimeline(ctx context.Context, evt contract.TimelineEvent) error {
	payload, err := encodeJSON(evt.Payload)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO timeline_events (id, task_id, type, payload_json, created_at) VALUES (?, ?, ?, ?, ?)`,
		evt.ID, evt.TaskID, evt.Type, payload, formatTime(evt.CreatedAt),
	)
	return err
}

func (s *Store) GetTimeline(ctx context.Context, taskID string) ([]contract.TimelineEvent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, task_id, type, payload_json, created_at FROM timeline_events WHERE task_id = ? ORDER BY created_at`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.TimelineEvent
	for rows.Next() {
		var evt contract.TimelineEvent
		var payload, created string
		if err := rows.Scan(&evt.ID, &evt.TaskID, &evt.Type, &payload, &created); err != nil {
			return nil, err
		}
		evt.CreatedAt, _ = parseTime(created)
		_ = decodeJSON(payload, &evt.Payload)
		out = append(out, evt)
	}
	return out, rows.Err()
}

func (s *Store) AppendMessage(ctx context.Context, msg *contract.TeamMessage) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (id, team_id, task_id, role, content, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.TeamID, msg.TaskID, string(msg.Role), msg.Content, formatTime(msg.CreatedAt),
	)
	return err
}

func (s *Store) ListMessages(ctx context.Context, taskID string) ([]contract.TeamMessage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, team_id, task_id, role, content, created_at FROM messages WHERE task_id = ? ORDER BY created_at`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.TeamMessage
	for rows.Next() {
		var m contract.TeamMessage
		var role, created string
		if err := rows.Scan(&m.ID, &m.TeamID, &m.TaskID, &role, &m.Content, &created); err != nil {
			return nil, err
		}
		m.Role = contract.MessageRole(role)
		m.CreatedAt, _ = parseTime(created)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) ListTasksByStatuses(ctx context.Context, statuses ...contract.TaskStatus) ([]contract.TeamTask, error) {
	if len(statuses) == 0 {
		return nil, nil
	}
	q := `SELECT id, team_id, content, status, close_reason, created_at, updated_at FROM tasks WHERE status IN (`
	args := make([]any, len(statuses))
	for i, st := range statuses {
		if i > 0 {
			q += ","
		}
		q += "?"
		args[i] = string(st)
	}
	q += `) ORDER BY updated_at`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func scanTasks(rows *sql.Rows) ([]contract.TeamTask, error) {
	var out []contract.TeamTask
	for rows.Next() {
		var t contract.TeamTask
		var status, closeReason, created, updated string
		if err := rows.Scan(&t.ID, &t.TeamID, &t.Content, &status, &closeReason, &created, &updated); err != nil {
			return nil, err
		}
		t.Status = contract.TaskStatus(status)
		t.CloseReason = contract.TaskCloseReason(closeReason)
		t.CreatedAt, _ = parseTime(created)
		t.UpdatedAt, _ = parseTime(updated)
		out = append(out, t)
	}
	return out, rows.Err()
}

func scanRun(scanner interface {
	Scan(dest ...any) error
}) (*contract.WorkerRun, error) {
	var run contract.WorkerRun
	var status, created, updated string
	if err := scanner.Scan(&run.ID, &run.TaskID, &run.DispatchID, &run.WorkerID, &status, &created, &updated); err != nil {
		return nil, err
	}
	run.Status = contract.RunStatus(status)
	run.CreatedAt, _ = parseTime(created)
	run.UpdatedAt, _ = parseTime(updated)
	return &run, nil
}

func (s *Store) GetReportByRunID(ctx context.Context, runID string) (*contract.Report, error) {
	var r contract.Report
	var actions, created, intent string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, run_id, task_id, worker_id, worker_name, content_markdown, intent, suggested_actions_json, created_at
		 FROM reports WHERE run_id = ?`, runID,
	).Scan(&r.ID, &r.RunID, &r.TaskID, &r.WorkerID, &r.WorkerName, &r.ContentMarkdown, &intent, &actions, &created)
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("report not found")
	}
	if err != nil {
		return nil, err
	}
	r.Intent = contract.ReportIntent(intent)
	r.CreatedAt, _ = parseTime(created)
	_ = decodeJSON(actions, &r.SuggestedActions)
	return &r, nil
}

func (s *Store) GetDispatchByRound(ctx context.Context, taskID string, round int) (*contract.Dispatch, error) {
	var d contract.Dispatch
	var created string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, task_id, worker_id, worker_name, intent, context_summary, round_num, created_at
		 FROM dispatches WHERE task_id = ? AND round_num = ?`, taskID, round,
	).Scan(&d.ID, &d.TaskID, &d.WorkerID, &d.WorkerName, &d.Intent, &d.ContextSummary, &d.Round, &created)
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("dispatch not found")
	}
	if err != nil {
		return nil, err
	}
	d.CreatedAt, _ = parseTime(created)
	return &d, nil
}

func (s *Store) GetRunByDispatchID(ctx context.Context, dispatchID string) (*contract.WorkerRun, error) {
	run, err := scanRun(s.db.QueryRowContext(ctx,
		`SELECT id, task_id, dispatch_id, worker_id, status, created_at, updated_at FROM worker_runs WHERE dispatch_id = ?`, dispatchID,
	))
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("run not found")
	}
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Store) LastUserMessage(ctx context.Context, taskID string) (string, error) {
	var content string
	err := s.db.QueryRowContext(ctx,
		`SELECT content FROM messages WHERE task_id = ? AND role = ? ORDER BY created_at DESC LIMIT 1`,
		taskID, string(contract.MessageRoleUser),
	).Scan(&content)
	if err == sql.ErrNoRows {
		return "", errs.NotFound("message not found")
	}
	return content, err
}

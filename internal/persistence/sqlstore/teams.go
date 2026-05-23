package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
	"danqing-teams/pkg/id"
)

func (s *Store) ListTeams(ctx context.Context) ([]contract.Team, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, created_at, updated_at FROM teams ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.Team
	for rows.Next() {
		var t contract.Team
		var created, updated string
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &created, &updated); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = parseTime(created)
		t.UpdatedAt, _ = parseTime(updated)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTeam(ctx context.Context, teamID string) (*contract.TeamDetail, error) {
	var t contract.Team
	var created, updated string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM teams WHERE id = ?`, teamID,
	).Scan(&t.ID, &t.Name, &t.Description, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("team not found")
	}
	if err != nil {
		return nil, err
	}
	t.CreatedAt, _ = parseTime(created)
	t.UpdatedAt, _ = parseTime(updated)

	var ctrl contract.TeamController
	err = s.db.QueryRowContext(ctx,
		`SELECT persona, system_prompt FROM team_controllers WHERE team_id = ?`, teamID,
	).Scan(&ctrl.Persona, &ctrl.SystemPrompt)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	workers, err := s.ListWorkers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	humans, err := s.ListHumans(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return &contract.TeamDetail{Team: t, Controller: ctrl, Workers: workers, Humans: humans}, nil
}

func (s *Store) CreateTeam(ctx context.Context, req contract.CreateTeamRequest) (*contract.TeamDetail, error) {
	tid := id.New()
	now := formatTime(nowUTC())
	t := contract.Team{ID: tid, Name: req.Name, Description: req.Description, CreatedAt: nowUTC(), UpdatedAt: nowUTC()}
	ctrl := contract.TeamController{
		Persona:      "负责理解用户意图，按 Worker 人设分派任务，汇总报告并规划 follow-up。",
		SystemPrompt: "你是 Team Controller，仅依据 Worker 人设匹配，不知道 Worker 的技能与 MCP Tool。",
	}
	err := withTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO teams (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			tid, req.Name, req.Description, now, now,
		); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO team_controllers (team_id, persona, system_prompt) VALUES (?, ?, ?)`,
			tid, ctrl.Persona, ctrl.SystemPrompt,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &contract.TeamDetail{Team: t, Controller: ctrl}, nil
}

func (s *Store) UpdateTeam(ctx context.Context, teamID string, req contract.UpdateTeamRequest) (*contract.Team, error) {
	t, err := s.getTeamRow(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	t.UpdatedAt = nowUTC()
	_, err = s.db.ExecContext(ctx,
		`UPDATE teams SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		t.Name, t.Description, formatTime(t.UpdatedAt), teamID,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Store) DeleteTeam(ctx context.Context, teamID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM teams WHERE id = ?`, teamID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errs.NotFound("team not found")
	}
	return nil
}

func (s *Store) getTeamRow(ctx context.Context, teamID string) (*contract.Team, error) {
	var t contract.Team
	var created, updated string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM teams WHERE id = ?`, teamID,
	).Scan(&t.ID, &t.Name, &t.Description, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("team not found")
	}
	if err != nil {
		return nil, err
	}
	t.CreatedAt, _ = parseTime(created)
	t.UpdatedAt, _ = parseTime(updated)
	return &t, nil
}

func (s *Store) ListPersonaCatalog(ctx context.Context, teamID string) ([]contract.WorkerPersonaCatalog, error) {
	workers, err := s.ListWorkers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	out := make([]contract.WorkerPersonaCatalog, len(workers))
	for i, w := range workers {
		out[i] = contract.WorkerPersonaCatalog{ID: w.ID, Name: w.Name, Persona: w.Persona}
	}
	return out, nil
}

func (s *Store) GetController(ctx context.Context, teamID string) (*contract.TeamController, error) {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return nil, err
	}
	var c contract.TeamController
	err := s.db.QueryRowContext(ctx,
		`SELECT persona, system_prompt FROM team_controllers WHERE team_id = ?`, teamID,
	).Scan(&c.Persona, &c.SystemPrompt)
	if err == sql.ErrNoRows {
		return &contract.TeamController{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) UpdateController(ctx context.Context, teamID string, c contract.TeamController) error {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO team_controllers (team_id, persona, system_prompt) VALUES (?, ?, ?)
		 ON CONFLICT(team_id) DO UPDATE SET persona = excluded.persona, system_prompt = excluded.system_prompt`,
		teamID, c.Persona, c.SystemPrompt,
	)
	return err
}

func (s *Store) ListWorkers(ctx context.Context, teamID string) ([]contract.WorkerAgent, error) {
	if workers, err := s.listWorkersFromAgents(ctx, teamID); err != nil {
		return nil, err
	} else if len(workers) > 0 {
		return workers, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, persona, skills_json, tools_json, kb_json FROM workers WHERE team_id = ?`, teamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.WorkerAgent
	for rows.Next() {
		w, err := scanWorker(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) GetWorker(ctx context.Context, teamID, workerID string) (*contract.WorkerAgent, error) {
	if w, err := s.getWorkerFromAgents(ctx, teamID, workerID); err == nil {
		return w, nil
	} else if !errors.Is(err, errs.ErrNotFound) {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, persona, skills_json, tools_json, kb_json FROM workers WHERE team_id = ? AND id = ?`,
		teamID, workerID,
	)
	w, err := scanWorkerRow(row)
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("worker not found")
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *Store) UpsertWorker(ctx context.Context, teamID string, worker *contract.WorkerAgent) error {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return err
	}
	if worker.ID == "" {
		worker.ID = id.New()
	}
	skills, _ := encodeJSON(worker.Skills)
	tools, _ := encodeJSON(worker.Tools)
	kb, _ := encodeJSON(worker.KnowledgeBase)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workers (id, team_id, name, persona, skills_json, tools_json, kb_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET name=excluded.name, persona=excluded.persona,
		   skills_json=excluded.skills_json, tools_json=excluded.tools_json, kb_json=excluded.kb_json`,
		worker.ID, teamID, worker.Name, worker.Persona, skills, tools, kb,
	)
	return err
}

func (s *Store) DeleteWorker(ctx context.Context, teamID, workerID string) error {
	if err := s.RemoveTeamAgent(ctx, teamID, workerID); err == nil {
		return nil
	} else if !errors.Is(err, errs.ErrNotFound) {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM workers WHERE team_id = ? AND id = ?`, teamID, workerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errs.NotFound("worker not found")
	}
	return nil
}

func (s *Store) GetWorkerPrivateProfile(ctx context.Context, teamID, workerID string) (*contract.WorkerPrivateProfile, error) {
	w, err := s.GetWorker(ctx, teamID, workerID)
	if err != nil {
		return nil, err
	}
	return &contract.WorkerPrivateProfile{
		WorkerID: w.ID, Skills: w.Skills, Tools: w.Tools, KnowledgeBase: w.KnowledgeBase,
	}, nil
}

func (s *Store) ListHumans(ctx context.Context, teamID string) ([]contract.HumanMember, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, display_name, email, role FROM humans WHERE team_id = ?`, teamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.HumanMember
	for rows.Next() {
		var h contract.HumanMember
		if err := rows.Scan(&h.ID, &h.DisplayName, &h.Email, &h.Role); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) AddHuman(ctx context.Context, teamID string, h contract.HumanMember) error {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return err
	}
	if h.ID == "" {
		h.ID = id.New()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO humans (id, team_id, display_name, email, role) VALUES (?, ?, ?, ?, ?)`,
		h.ID, teamID, h.DisplayName, h.Email, h.Role,
	)
	return err
}

func scanWorker(rows *sql.Rows) (contract.WorkerAgent, error) {
	var w contract.WorkerAgent
	var skills, tools, kb string
	if err := rows.Scan(&w.ID, &w.Name, &w.Persona, &skills, &tools, &kb); err != nil {
		return w, err
	}
	_ = decodeJSON(skills, &w.Skills)
	_ = decodeJSON(tools, &w.Tools)
	_ = decodeJSON(kb, &w.KnowledgeBase)
	return w, nil
}

func scanWorkerRow(row *sql.Row) (contract.WorkerAgent, error) {
	var w contract.WorkerAgent
	var skills, tools, kb string
	if err := row.Scan(&w.ID, &w.Name, &w.Persona, &skills, &tools, &kb); err != nil {
		return w, err
	}
	_ = decodeJSON(skills, &w.Skills)
	_ = decodeJSON(tools, &w.Tools)
	_ = decodeJSON(kb, &w.KnowledgeBase)
	return w, nil
}

func nowUTC() time.Time { return time.Now().UTC() }

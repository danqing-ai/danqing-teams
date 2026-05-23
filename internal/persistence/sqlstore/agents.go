package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
)

func agentToWorker(a contract.Agent) contract.WorkerAgent {
	return contract.WorkerAgent{
		ID:            a.ID,
		Name:          a.Name,
		Persona:       a.Description,
		Skills:        a.Skills,
		Tools:         a.Tools,
		KnowledgeBase: a.KnowledgeBase,
	}
}

func sanitizeAgentForResponse(a *contract.Agent) {
	if a.LLM.APIKey != "" {
		a.LLM.HasAPIKey = true
		a.LLM.APIKey = ""
	}
}

func (s *Store) ListAgents(ctx context.Context, role contract.AgentRole) ([]contract.Agent, error) {
	q := `SELECT id, name, description, role, llm_url, llm_api_key, default_model, all_models_json,
		system_prompt, min_function_calling_rounds, skills_json, tools_json, kb_json, created_at, updated_at
		FROM agents`
	var args []any
	if role != "" {
		q += ` WHERE role = ?`
		args = append(args, string(role))
	}
	q += ` ORDER BY name`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		sanitizeAgentForResponse(&a)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAgent(ctx context.Context, agentID string) (*contract.Agent, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, role, llm_url, llm_api_key, default_model, all_models_json,
		 system_prompt, min_function_calling_rounds, skills_json, tools_json, kb_json, created_at, updated_at
		 FROM agents WHERE id = ?`, agentID,
	)
	a, err := scanAgentRow(row)
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("agent not found")
	}
	if err != nil {
		return nil, err
	}
	sanitizeAgentForResponse(&a)
	return &a, nil
}

func (s *Store) CreateAgent(ctx context.Context, req contract.CreateAgentRequest) (*contract.Agent, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errs.BadRequest("agent name required")
	}
	agentID, err := contract.ParseAgentID(req.ID)
	if err != nil {
		return nil, err
	}
	if req.Role == "" {
		req.Role = contract.AgentRoleTeamWorker
	}
	if _, err := s.getAgentRow(ctx, agentID); err == nil {
		return nil, errs.BadRequest("agent id already exists")
	} else if !errors.Is(err, errs.ErrNotFound) {
		return nil, err
	}
	now := nowUTC()
	a := contract.Agent{
		ID:                       agentID,
		Name:                     strings.TrimSpace(req.Name),
		Description:              req.Description,
		Role:                     req.Role,
		LLM:                      req.LLM,
		SystemPrompt:             req.SystemPrompt,
		MinFunctionCallingRounds: req.MinFunctionCallingRounds,
		Skills:                   req.Skills,
		Tools:                    req.Tools,
		KnowledgeBase:            req.KnowledgeBase,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if a.MinFunctionCallingRounds <= 0 {
		a.MinFunctionCallingRounds = 1
	}
	if err := s.insertAgent(ctx, &a); err != nil {
		return nil, err
	}
	sanitizeAgentForResponse(&a)
	return &a, nil
}

func (s *Store) UpdateAgent(ctx context.Context, agentID string, req contract.UpdateAgentRequest) (*contract.Agent, error) {
	a, err := s.getAgentRow(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		a.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		a.Description = *req.Description
	}
	if req.Role != nil {
		a.Role = *req.Role
	}
	if req.LLM != nil {
		if req.LLM.URL != "" || req.LLM.DefaultModel != "" || len(req.LLM.AllModels) > 0 {
			a.LLM.URL = req.LLM.URL
			a.LLM.DefaultModel = req.LLM.DefaultModel
			a.LLM.AllModels = req.LLM.AllModels
		}
		if req.LLM.APIKey != "" {
			a.LLM.APIKey = req.LLM.APIKey
		}
	}
	if req.SystemPrompt != nil {
		a.SystemPrompt = *req.SystemPrompt
	}
	if req.MinFunctionCallingRounds != nil {
		a.MinFunctionCallingRounds = *req.MinFunctionCallingRounds
	}
	if req.Skills != nil {
		a.Skills = *req.Skills
	}
	if req.Tools != nil {
		a.Tools = *req.Tools
	}
	if req.KnowledgeBase != nil {
		a.KnowledgeBase = *req.KnowledgeBase
	}
	a.UpdatedAt = nowUTC()
	if err := s.updateAgentRow(ctx, a); err != nil {
		return nil, err
	}
	sanitizeAgentForResponse(a)
	return a, nil
}

func (s *Store) DeleteAgent(ctx context.Context, agentID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM agents WHERE id = ?`, agentID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errs.NotFound("agent not found")
	}
	return nil
}

func (s *Store) ListTeamAgentMembers(ctx context.Context, teamID string) ([]contract.Agent, error) {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT a.id, a.name, a.description, a.role, a.llm_url, a.llm_api_key, a.default_model, a.all_models_json,
		 a.system_prompt, a.min_function_calling_rounds, a.skills_json, a.tools_json, a.kb_json, a.created_at, a.updated_at
		 FROM team_agents ta JOIN agents a ON a.id = ta.agent_id
		 WHERE ta.team_id = ? AND a.role = ?
		 ORDER BY a.name`, teamID, string(contract.AgentRoleTeamWorker),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		sanitizeAgentForResponse(&a)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) AddTeamAgent(ctx context.Context, teamID, agentID string) error {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return err
	}
	a, err := s.getAgentRow(ctx, agentID)
	if err != nil {
		return err
	}
	if a.Role != contract.AgentRoleTeamWorker {
		return errs.BadRequest("only team-worker agents can join a team")
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO team_agents (team_id, agent_id) VALUES (?, ?)`, teamID, agentID,
	)
	return err
}

func (s *Store) RemoveTeamAgent(ctx context.Context, teamID, agentID string) error {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM team_agents WHERE team_id = ? AND agent_id = ?`, teamID, agentID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errs.NotFound("agent not in team")
	}
	return nil
}

func (s *Store) IsTeamAgentMember(ctx context.Context, teamID, agentID string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM team_agents WHERE team_id = ? AND agent_id = ?`, teamID, agentID,
	).Scan(&n)
	return n > 0, err
}

func (s *Store) listWorkersFromAgents(ctx context.Context, teamID string) ([]contract.WorkerAgent, error) {
	members, err := s.ListTeamAgentMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	out := make([]contract.WorkerAgent, len(members))
	for i, a := range members {
		out[i] = agentToWorker(a)
	}
	return out, nil
}

func (s *Store) getWorkerFromAgents(ctx context.Context, teamID, workerID string) (*contract.WorkerAgent, error) {
	ok, err := s.IsTeamAgentMember(ctx, teamID, workerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.NotFound("worker not found")
	}
	a, err := s.getAgentRow(ctx, workerID)
	if err != nil {
		return nil, err
	}
	w := agentToWorker(*a)
	return &w, nil
}

func (s *Store) insertAgent(ctx context.Context, a *contract.Agent) error {
	skills, _ := encodeJSON(a.Skills)
	tools, _ := encodeJSON(a.Tools)
	kb, _ := encodeJSON(a.KnowledgeBase)
	models, _ := encodeJSON(a.LLM.AllModels)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO agents (id, name, description, role, llm_url, llm_api_key, default_model, all_models_json,
		 system_prompt, min_function_calling_rounds, skills_json, tools_json, kb_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Name, a.Description, string(a.Role), a.LLM.URL, a.LLM.APIKey, a.LLM.DefaultModel, models,
		a.SystemPrompt, a.MinFunctionCallingRounds, skills, tools, kb, formatTime(a.CreatedAt), formatTime(a.UpdatedAt),
	)
	return err
}

func (s *Store) updateAgentRow(ctx context.Context, a *contract.Agent) error {
	skills, _ := encodeJSON(a.Skills)
	tools, _ := encodeJSON(a.Tools)
	kb, _ := encodeJSON(a.KnowledgeBase)
	models, _ := encodeJSON(a.LLM.AllModels)
	_, err := s.db.ExecContext(ctx,
		`UPDATE agents SET name=?, description=?, role=?, llm_url=?, llm_api_key=?, default_model=?, all_models_json=?,
		 system_prompt=?, min_function_calling_rounds=?, skills_json=?, tools_json=?, kb_json=?, updated_at=?
		 WHERE id=?`,
		a.Name, a.Description, string(a.Role), a.LLM.URL, a.LLM.APIKey, a.LLM.DefaultModel, models,
		a.SystemPrompt, a.MinFunctionCallingRounds, skills, tools, kb, formatTime(a.UpdatedAt), a.ID,
	)
	return err
}

func (s *Store) getAgentRow(ctx context.Context, agentID string) (*contract.Agent, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, role, llm_url, llm_api_key, default_model, all_models_json,
		 system_prompt, min_function_calling_rounds, skills_json, tools_json, kb_json, created_at, updated_at
		 FROM agents WHERE id = ?`, agentID,
	)
	a, err := scanAgentRow(row)
	if err == sql.ErrNoRows {
		return nil, errs.NotFound("agent not found")
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

type agentScanner interface {
	Scan(dest ...any) error
}

func scanAgent(rows agentScanner) (contract.Agent, error) {
	var a contract.Agent
	var role, models, skills, tools, kb, created, updated string
	if err := rows.Scan(
		&a.ID, &a.Name, &a.Description, &role, &a.LLM.URL, &a.LLM.APIKey, &a.LLM.DefaultModel, &models,
		&a.SystemPrompt, &a.MinFunctionCallingRounds, &skills, &tools, &kb, &created, &updated,
	); err != nil {
		return a, err
	}
	a.Role = contract.AgentRole(role)
	_ = decodeJSON(models, &a.LLM.AllModels)
	_ = decodeJSON(skills, &a.Skills)
	_ = decodeJSON(tools, &a.Tools)
	_ = decodeJSON(kb, &a.KnowledgeBase)
	a.CreatedAt, _ = parseTime(created)
	a.UpdatedAt, _ = parseTime(updated)
	return a, nil
}

func scanAgentRow(row *sql.Row) (contract.Agent, error) {
	return scanAgent(row)
}

func (s *Store) MigrateLegacyWorkersToAgents(ctx context.Context) error {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM agents`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, team_id, name, persona, skills_json, tools_json, kb_json FROM workers`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	type legacy struct {
		id, teamID, name, persona, skills, tools, kb string
	}
	var items []legacy
	for rows.Next() {
		var w legacy
		if err := rows.Scan(&w.id, &w.teamID, &w.name, &w.persona, &w.skills, &w.tools, &w.kb); err != nil {
			return err
		}
		items = append(items, w)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := nowUTC()
	for _, w := range items {
		a := contract.Agent{
			ID:          w.id,
			Name:        w.name,
			Description: w.persona,
			Role:        contract.AgentRoleTeamWorker,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		_ = decodeJSON(w.skills, &a.Skills)
		_ = decodeJSON(w.tools, &a.Tools)
		_ = decodeJSON(w.kb, &a.KnowledgeBase)
		if err := s.insertAgent(ctx, &a); err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO team_agents (team_id, agent_id) VALUES (?, ?)`, w.teamID, w.id,
		); err != nil {
			return err
		}
	}
	return nil
}

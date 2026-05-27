package sqlstore

import (
	"context"
	"errors"
	"strings"

	"danqing-teams/internal/domain/model"
	"danqing-teams/pkg/errs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func agentToWorker(a model.Agent) model.WorkerAgent {
	return model.WorkerAgent{
		ID: a.ID, Name: a.Name, Persona: a.Description,
		Skills: a.Skills, Tools: a.Tools, KnowledgeBase: a.KnowledgeBase,
	}
}

func sanitizeAgentForResponse(a *model.Agent) {
	if a.LLM.APIKey != "" {
		a.LLM.HasAPIKey = true
		a.LLM.APIKey = ""
	}
}

func (s *Store) ListAgents(ctx context.Context, role model.AgentRole) ([]model.Agent, error) {
	q := s.dbWithCtx(ctx)
	if role != "" {
		q = q.Where("role = ?", role)
	}
	var rows []agentRow
	if err := q.Order("name").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]model.Agent, len(rows))
	for i, r := range rows {
		out[i] = agentFromRow(r)
		sanitizeAgentForResponse(&out[i])
	}
	return out, nil
}

func (s *Store) GetAgent(ctx context.Context, agentID string) (*model.Agent, error) {
	a, err := s.getAgentRow(ctx, agentID)
	if err != nil {
		return nil, err
	}
	sanitizeAgentForResponse(a)
	return a, nil
}

func (s *Store) CreateAgent(ctx context.Context, req model.CreateAgentRequest) (*model.Agent, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errs.BadRequest("agent name required")
	}
	agentID, err := model.ParseAgentID(req.ID)
	if err != nil {
		return nil, err
	}
	if req.Role == "" {
		req.Role = model.AgentRoleTeamWorker
	}
	if _, err := s.getAgentRow(ctx, agentID); err == nil {
		return nil, errs.BadRequest("agent id already exists")
	} else if !errors.Is(err, errs.ErrNotFound) {
		return nil, err
	}
	now := nowUTC()
	a := model.Agent{
		ID: agentID, Name: strings.TrimSpace(req.Name), Description: req.Description, Role: req.Role,
		LLM: req.LLM, SystemPrompt: req.SystemPrompt, MinFunctionCallingRounds: req.MinFunctionCallingRounds,
		Skills: req.Skills, Tools: req.Tools, KnowledgeBase: req.KnowledgeBase,
		CreatedAt: now, UpdatedAt: now,
	}
	if a.MinFunctionCallingRounds <= 0 {
		a.MinFunctionCallingRounds = 1
	}
	row := agentToRow(&a)
	if err := s.dbWithCtx(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	sanitizeAgentForResponse(&a)
	return &a, nil
}

func (s *Store) UpdateAgent(ctx context.Context, agentID string, req model.UpdateAgentRequest) (*model.Agent, error) {
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
	row := agentToRow(a)
	if err := s.dbWithCtx(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	sanitizeAgentForResponse(a)
	return a, nil
}

func (s *Store) DeleteAgent(ctx context.Context, agentID string) error {
	res := s.dbWithCtx(ctx).Delete(&agentRow{}, "id = ?", agentID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("agent not found")
	}
	return nil
}

func (s *Store) ListTeamAgentMembers(ctx context.Context, teamID string) ([]model.Agent, error) {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return nil, err
	}
	var rows []agentRow
	err := s.dbWithCtx(ctx).
		Table("team_agents ta").
		Select("a.*").
		Joins("JOIN agents a ON a.id = ta.agent_id").
		Where("ta.team_id = ? AND a.role = ?", teamID, model.AgentRoleTeamWorker).
		Order("a.name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]model.Agent, len(rows))
	for i, r := range rows {
		out[i] = agentFromRow(r)
		sanitizeAgentForResponse(&out[i])
	}
	return out, nil
}

func (s *Store) AddTeamAgent(ctx context.Context, teamID, agentID string) error {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return err
	}
	a, err := s.getAgentRow(ctx, agentID)
	if err != nil {
		return err
	}
	if a.Role != model.AgentRoleTeamWorker {
		return errs.BadRequest("only team-worker agents can join a team")
	}
	return s.dbWithCtx(ctx).Clauses(clause.Insert{Modifier: "OR IGNORE"}).Create(&teamAgentRow{
		TeamID: teamID, AgentID: agentID,
	}).Error
}

func (s *Store) RemoveTeamAgent(ctx context.Context, teamID, agentID string) error {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return err
	}
	res := s.dbWithCtx(ctx).Where("team_id = ? AND agent_id = ?", teamID, agentID).Delete(&teamAgentRow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("agent not in team")
	}
	return nil
}

func (s *Store) IsTeamAgentMember(ctx context.Context, teamID, agentID string) (bool, error) {
	var n int64
	err := s.dbWithCtx(ctx).Model(&teamAgentRow{}).
		Where("team_id = ? AND agent_id = ?", teamID, agentID).Count(&n).Error
	return n > 0, err
}

func (s *Store) listWorkersFromAgents(ctx context.Context, teamID string) ([]model.WorkerAgent, error) {
	members, err := s.ListTeamAgentMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	out := make([]model.WorkerAgent, len(members))
	for i, a := range members {
		out[i] = agentToWorker(a)
	}
	return out, nil
}

func (s *Store) getWorkerFromAgents(ctx context.Context, teamID, workerID string) (*model.WorkerAgent, error) {
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

func (s *Store) getAgentRow(ctx context.Context, agentID string) (*model.Agent, error) {
	var r agentRow
	if err := s.dbWithCtx(ctx).First(&r, "id = ?", agentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("agent not found")
		}
		return nil, err
	}
	a := agentFromRow(r)
	return &a, nil
}

func (s *Store) MigrateLegacyWorkersToAgents(ctx context.Context) error {
	var n int64
	if err := s.dbWithCtx(ctx).Model(&agentRow{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var workers []workerRow
	if err := s.dbWithCtx(ctx).Find(&workers).Error; err != nil {
		return err
	}
	now := nowUTC()
	for _, w := range workers {
		a := model.Agent{
			ID: w.ID, Name: w.Name, Description: w.Persona, Role: model.AgentRoleTeamWorker,
			Skills: w.Skills, Tools: w.Tools, KnowledgeBase: w.KB,
			CreatedAt: now, UpdatedAt: now,
		}
		row := agentToRow(&a)
	if err := s.dbWithCtx(ctx).Create(&row).Error; err != nil {
			return err
		}
		if err := s.dbWithCtx(ctx).Clauses(clause.Insert{Modifier: "OR IGNORE"}).Create(&teamAgentRow{
			TeamID: w.TeamID, AgentID: w.ID,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

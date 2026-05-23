package sqlstore

import (
	"context"
	"errors"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
	"danqing-teams/pkg/id"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) ListTeams(ctx context.Context) ([]contract.Team, error) {
	var rows []teamRow
	if err := s.dbWithCtx(ctx).Order("created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]contract.Team, len(rows))
	for i, r := range rows {
		out[i] = teamFromRow(r)
	}
	return out, nil
}

func (s *Store) GetTeam(ctx context.Context, teamID string) (*contract.TeamDetail, error) {
	t, err := s.getTeamRow(ctx, teamID)
	if err != nil {
		return nil, err
	}
	var ctrl teamControllerRow
	if err := s.dbWithCtx(ctx).First(&ctrl, "team_id = ?", teamID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
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
	return &contract.TeamDetail{
		Team: *t,
		Controller: contract.TeamController{Persona: ctrl.Persona, SystemPrompt: ctrl.SystemPrompt},
		Workers: workers, Humans: humans,
	}, nil
}

func (s *Store) CreateTeam(ctx context.Context, req contract.CreateTeamRequest) (*contract.TeamDetail, error) {
	tid := id.New()
	now := nowUTC()
	t := contract.Team{ID: tid, Name: req.Name, Description: req.Description, CreatedAt: now, UpdatedAt: now}
	ctrl := contract.TeamController{
		Persona:      "负责理解用户意图，按 Worker 人设分派任务，汇总报告并规划 follow-up。",
		SystemPrompt: "你是 Team Controller，仅依据 Worker 人设匹配，不知道 Worker 的技能与 MCP Tool。",
	}
	err := s.dbWithCtx(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&teamRow{
			ID: tid, Name: req.Name, Description: req.Description, CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&teamControllerRow{TeamID: tid, Persona: ctrl.Persona, SystemPrompt: ctrl.SystemPrompt}).Error
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
	res := s.dbWithCtx(ctx).Model(&teamRow{}).Where("id = ?", teamID).Updates(map[string]any{
		"name": t.Name, "description": t.Description, "updated_at": t.UpdatedAt,
	})
	if res.Error != nil {
		return nil, res.Error
	}
	return t, nil
}

func (s *Store) DeleteTeam(ctx context.Context, teamID string) error {
	res := s.dbWithCtx(ctx).Delete(&teamRow{}, "id = ?", teamID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("team not found")
	}
	return nil
}

func (s *Store) getTeamRow(ctx context.Context, teamID string) (*contract.Team, error) {
	var r teamRow
	if err := s.dbWithCtx(ctx).First(&r, "id = ?", teamID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("team not found")
		}
		return nil, err
	}
	t := teamFromRow(r)
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
	var ctrl teamControllerRow
	if err := s.dbWithCtx(ctx).First(&ctrl, "team_id = ?", teamID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &contract.TeamController{}, nil
		}
		return nil, err
	}
	return &contract.TeamController{Persona: ctrl.Persona, SystemPrompt: ctrl.SystemPrompt}, nil
}

func (s *Store) UpdateController(ctx context.Context, teamID string, c contract.TeamController) error {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return err
	}
	row := teamControllerRow{TeamID: teamID, Persona: c.Persona, SystemPrompt: c.SystemPrompt}
	return s.dbWithCtx(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "team_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"persona", "system_prompt"}),
	}).Create(&row).Error
}

func (s *Store) ListWorkers(ctx context.Context, teamID string) ([]contract.WorkerAgent, error) {
	if workers, err := s.listWorkersFromAgents(ctx, teamID); err != nil {
		return nil, err
	} else if len(workers) > 0 {
		return workers, nil
	}
	var rows []workerRow
	if err := s.dbWithCtx(ctx).Where("team_id = ?", teamID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]contract.WorkerAgent, len(rows))
	for i, r := range rows {
		out[i] = workerFromRow(r)
	}
	return out, nil
}

func (s *Store) GetWorker(ctx context.Context, teamID, workerID string) (*contract.WorkerAgent, error) {
	if w, err := s.getWorkerFromAgents(ctx, teamID, workerID); err == nil {
		return w, nil
	} else if !errors.Is(err, errs.ErrNotFound) {
		return nil, err
	}
	var r workerRow
	if err := s.dbWithCtx(ctx).Where("team_id = ? AND id = ?", teamID, workerID).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("worker not found")
		}
		return nil, err
	}
	w := workerFromRow(r)
	return &w, nil
}

func (s *Store) UpsertWorker(ctx context.Context, teamID string, worker *contract.WorkerAgent) error {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return err
	}
	if worker.ID == "" {
		worker.ID = id.New()
	}
	row := workerRow{
		ID: worker.ID, TeamID: teamID, Name: worker.Name, Persona: worker.Persona,
		Skills: worker.Skills, Tools: worker.Tools, KB: worker.KnowledgeBase,
	}
	return s.dbWithCtx(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "persona", "skills_json", "tools_json", "kb_json"}),
	}).Create(&row).Error
}

func (s *Store) DeleteWorker(ctx context.Context, teamID, workerID string) error {
	if err := s.RemoveTeamAgent(ctx, teamID, workerID); err == nil {
		return nil
	} else if !errors.Is(err, errs.ErrNotFound) {
		return err
	}
	res := s.dbWithCtx(ctx).Where("team_id = ? AND id = ?", teamID, workerID).Delete(&workerRow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
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
	var rows []humanRow
	if err := s.dbWithCtx(ctx).Where("team_id = ?", teamID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]contract.HumanMember, len(rows))
	for i, r := range rows {
		out[i] = contract.HumanMember{ID: r.ID, DisplayName: r.DisplayName, Email: r.Email, Role: r.Role}
	}
	return out, nil
}

func (s *Store) AddHuman(ctx context.Context, teamID string, h contract.HumanMember) error {
	if _, err := s.getTeamRow(ctx, teamID); err != nil {
		return err
	}
	if h.ID == "" {
		h.ID = id.New()
	}
	return s.dbWithCtx(ctx).Create(&humanRow{
		ID: h.ID, TeamID: teamID, DisplayName: h.DisplayName, Email: h.Email, Role: h.Role,
	}).Error
}

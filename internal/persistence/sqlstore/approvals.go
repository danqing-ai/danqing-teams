package sqlstore

import (
	"context"
	"errors"

	"danqing-teams/internal/domain/model"
	"danqing-teams/pkg/errs"
	"danqing-teams/pkg/id"
	"gorm.io/gorm"
)

func (s *Store) Create(ctx context.Context, req *model.ApprovalRequest) error {
	return s.createApproval(ctx, req)
}

func (s *Store) createApproval(ctx context.Context, req *model.ApprovalRequest) error {
	return s.dbWithCtx(ctx).Create(&approvalRow{
		ID: req.ID, TeamID: req.TeamID, TaskID: req.TaskID, RunID: req.RunID,
		Summary: req.Summary, HighRiskItems: req.HighRiskItems,
		Status: req.Status, Comment: req.Comment,
		CreatedAt: req.CreatedAt, UpdatedAt: req.UpdatedAt,
	}).Error
}

func (s *Store) Get(ctx context.Context, teamID, approvalID string) (*model.ApprovalRequest, error) {
	return s.GetApproval(ctx, teamID, approvalID)
}

func (s *Store) GetApproval(ctx context.Context, _, approvalID string) (*model.ApprovalRequest, error) {
	var r approvalRow
	if err := s.dbWithCtx(ctx).First(&r, "id = ?", approvalID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("approval not found")
		}
		return nil, err
	}
	a := approvalFromRow(r)
	return &a, nil
}

func (s *Store) Update(ctx context.Context, req *model.ApprovalRequest) error {
	return s.UpdateApproval(ctx, req)
}

func (s *Store) UpdateApproval(ctx context.Context, req *model.ApprovalRequest) error {
	res := s.dbWithCtx(ctx).Model(&approvalRow{}).Where("id = ?", req.ID).Updates(map[string]any{
		"status": req.Status, "comment": req.Comment,
		"high_risk_json": req.HighRiskItems, "summary": req.Summary, "updated_at": req.UpdatedAt,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("approval not found")
	}
	return nil
}

func (s *Store) List(ctx context.Context, teamID string, status model.ApprovalStatus) ([]model.ApprovalRequest, error) {
	return s.ListApprovals(ctx, teamID, status)
}

func (s *Store) ListApprovals(ctx context.Context, teamID string, status model.ApprovalStatus) ([]model.ApprovalRequest, error) {
	q := s.dbWithCtx(ctx).Where("team_id = ?", teamID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []approvalRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]model.ApprovalRequest, len(rows))
	for i, r := range rows {
		out[i] = approvalFromRow(r)
	}
	return out, nil
}

func (s *Store) GetByRunID(ctx context.Context, runID string) (*model.ApprovalRequest, error) {
	return s.GetApprovalByRunID(ctx, runID)
}

func (s *Store) GetApprovalByRunID(ctx context.Context, runID string) (*model.ApprovalRequest, error) {
	var r approvalRow
	if err := s.dbWithCtx(ctx).First(&r, "run_id = ?", runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("approval not found")
		}
		return nil, err
	}
	a := approvalFromRow(r)
	return &a, nil
}

func (s *Store) ListTodos(ctx context.Context, teamID, taskID string) ([]model.TodoItem, error) {
	q := s.dbWithCtx(ctx).Where("team_id = ?", teamID)
	if taskID != "" {
		q = q.Where("task_id = ?", taskID)
	}
	var rows []todoRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]model.TodoItem, len(rows))
	for i, r := range rows {
		out[i] = model.TodoItem{
			ID: r.ID, TeamID: r.TeamID, TaskID: r.TaskID,
			Title: r.Title, Done: r.Done, CreatedAt: r.CreatedAt,
		}
	}
	return out, nil
}

func (s *Store) CreateTodo(ctx context.Context, teamID string, item model.TodoItem) (*model.TodoItem, error) {
	if item.ID == "" {
		item.ID = id.New()
	}
	item.TeamID = teamID
	item.CreatedAt = nowUTC()
	if err := s.dbWithCtx(ctx).Create(&todoRow{
		ID: item.ID, TeamID: teamID, TaskID: item.TaskID,
		Title: item.Title, Done: item.Done, CreatedAt: item.CreatedAt,
	}).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) UpdateTodo(ctx context.Context, teamID, todoID string, done bool) (*model.TodoItem, error) {
	res := s.dbWithCtx(ctx).Model(&todoRow{}).Where("team_id = ? AND id = ?", teamID, todoID).Update("done", done)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
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

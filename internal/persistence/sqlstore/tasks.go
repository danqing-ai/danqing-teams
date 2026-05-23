package sqlstore

import (
	"context"
	"errors"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) ListTasks(ctx context.Context, teamID string, status contract.TaskStatus) ([]contract.TeamTask, error) {
	q := s.dbWithCtx(ctx).Where("team_id = ?", teamID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []taskRow
	if err := q.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return tasksFromRows(rows), nil
}

func (s *Store) GetTask(ctx context.Context, teamID, taskID string) (*contract.TeamTask, error) {
	var r taskRow
	if err := s.dbWithCtx(ctx).First(&r, "id = ?", taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("task not found")
		}
		return nil, err
	}
	if teamID != "" && r.TeamID != teamID {
		return nil, errs.NotFound("task not found")
	}
	t := taskFromRow(r)
	return &t, nil
}

func (s *Store) CreateTask(ctx context.Context, task *contract.TeamTask) error {
	return s.dbWithCtx(ctx).Create(&taskRow{
		ID: task.ID, TeamID: task.TeamID, Content: task.Content,
		Status: task.Status, CloseReason: task.CloseReason,
		CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt,
	}).Error
}

func (s *Store) UpdateTaskStatus(ctx context.Context, _, taskID string, status contract.TaskStatus) error {
	res := s.dbWithCtx(ctx).Model(&taskRow{}).Where("id = ?", taskID).Updates(map[string]any{
		"status": status, "updated_at": nowUTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("task not found")
	}
	return nil
}

func (s *Store) UpdateTaskClosure(ctx context.Context, _, taskID string, status contract.TaskStatus, reason contract.TaskCloseReason) error {
	res := s.dbWithCtx(ctx).Model(&taskRow{}).Where("id = ?", taskID).Updates(map[string]any{
		"status": status, "close_reason": reason, "updated_at": nowUTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("task not found")
	}
	return nil
}

func (s *Store) SaveDispatch(ctx context.Context, d *contract.Dispatch) error {
	return s.dbWithCtx(ctx).Create(&dispatchRow{
		ID: d.ID, TaskID: d.TaskID, WorkerID: d.WorkerID, WorkerName: d.WorkerName,
		Intent: d.Intent, ContextSummary: d.ContextSummary, Round: d.Round, CreatedAt: d.CreatedAt,
	}).Error
}

func (s *Store) ListDispatches(ctx context.Context, taskID string) ([]contract.Dispatch, error) {
	var rows []dispatchRow
	if err := s.dbWithCtx(ctx).Where("task_id = ?", taskID).Order("created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]contract.Dispatch, len(rows))
	for i, r := range rows {
		out[i] = dispatchFromRow(r)
	}
	return out, nil
}

func (s *Store) SaveRun(ctx context.Context, run *contract.WorkerRun) error {
	return s.dbWithCtx(ctx).Create(&workerRunRow{
		ID: run.ID, TaskID: run.TaskID, DispatchID: run.DispatchID,
		WorkerID: run.WorkerID, Status: run.Status,
		CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt,
	}).Error
}

func (s *Store) GetRun(ctx context.Context, runID string) (*contract.WorkerRun, error) {
	var r workerRunRow
	if err := s.dbWithCtx(ctx).First(&r, "id = ?", runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("run not found")
		}
		return nil, err
	}
	run := runFromRow(r)
	plan, err := s.GetPlan(ctx, runID)
	if err == nil {
		run.Plan = plan
	}
	return &run, nil
}

func (s *Store) UpdateRun(ctx context.Context, run *contract.WorkerRun) error {
	res := s.dbWithCtx(ctx).Model(&workerRunRow{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status": run.Status, "updated_at": nowUTC(),
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.NotFound("run not found")
	}
	return nil
}

func (s *Store) ListRuns(ctx context.Context, taskID string) ([]contract.WorkerRun, error) {
	var rows []workerRunRow
	if err := s.dbWithCtx(ctx).Where("task_id = ?", taskID).Order("created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]contract.WorkerRun, len(rows))
	for i, r := range rows {
		run := runFromRow(r)
		plan, err := s.GetPlan(ctx, run.ID)
		if err == nil {
			run.Plan = plan
		}
		out[i] = run
	}
	return out, nil
}

func (s *Store) SavePlan(ctx context.Context, plan *contract.ExecutionPlan) error {
	row := executionPlanRow{
		RunID: plan.RunID, SkillIDs: plan.SkillIDs, ToolIDs: plan.ToolIDs,
		Rationale: plan.Rationale, EvaluatedRisk: plan.EvaluatedRisk, HighRiskItems: plan.HighRiskItems,
	}
	return s.dbWithCtx(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "run_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"skill_ids_json", "tool_ids_json", "rationale", "evaluated_risk", "high_risk_json",
		}),
	}).Create(&row).Error
}

func (s *Store) GetPlan(ctx context.Context, runID string) (*contract.ExecutionPlan, error) {
	var r executionPlanRow
	if err := s.dbWithCtx(ctx).First(&r, "run_id = ?", runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("plan not found")
		}
		return nil, err
	}
	plan := planFromRow(r)
	return &plan, nil
}

func (s *Store) SaveReport(ctx context.Context, r *contract.Report) error {
	row := reportRow{
		ID: r.ID, RunID: r.RunID, TaskID: r.TaskID, WorkerID: r.WorkerID, WorkerName: r.WorkerName,
		ContentMarkdown: r.ContentMarkdown, Intent: r.Intent,
		SuggestedActions: r.SuggestedActions, CreatedAt: r.CreatedAt,
	}
	return s.dbWithCtx(ctx).Clauses(clause.Insert{Modifier: "OR IGNORE"}).Create(&row).Error
}

func (s *Store) ListReports(ctx context.Context, taskID string) ([]contract.Report, error) {
	var rows []reportRow
	if err := s.dbWithCtx(ctx).Where("task_id = ?", taskID).Order("created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]contract.Report, len(rows))
	for i, r := range rows {
		out[i] = reportFromRow(r)
	}
	return out, nil
}

func (s *Store) AppendTimeline(ctx context.Context, evt contract.TimelineEvent) error {
	return s.dbWithCtx(ctx).Create(&timelineEventRow{
		ID: evt.ID, TaskID: evt.TaskID, Type: evt.Type, Payload: evt.Payload, CreatedAt: evt.CreatedAt,
	}).Error
}

func (s *Store) GetTimeline(ctx context.Context, taskID string) ([]contract.TimelineEvent, error) {
	var rows []timelineEventRow
	if err := s.dbWithCtx(ctx).Where("task_id = ?", taskID).Order("created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]contract.TimelineEvent, len(rows))
	for i, r := range rows {
		out[i] = contract.TimelineEvent{
			ID: r.ID, TaskID: r.TaskID, Type: r.Type, Payload: r.Payload, CreatedAt: r.CreatedAt,
		}
	}
	return out, nil
}

func (s *Store) AppendMessage(ctx context.Context, msg *contract.TeamMessage) error {
	return s.dbWithCtx(ctx).Create(&messageRow{
		ID: msg.ID, TeamID: msg.TeamID, TaskID: msg.TaskID,
		Role: msg.Role, Content: msg.Content, CreatedAt: msg.CreatedAt,
	}).Error
}

func (s *Store) ListMessages(ctx context.Context, taskID string) ([]contract.TeamMessage, error) {
	var rows []messageRow
	if err := s.dbWithCtx(ctx).Where("task_id = ?", taskID).Order("created_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]contract.TeamMessage, len(rows))
	for i, r := range rows {
		out[i] = contract.TeamMessage{
			ID: r.ID, TeamID: r.TeamID, TaskID: r.TaskID,
			Role: r.Role, Content: r.Content, CreatedAt: r.CreatedAt,
		}
	}
	return out, nil
}

func (s *Store) ListTasksByStatuses(ctx context.Context, statuses ...contract.TaskStatus) ([]contract.TeamTask, error) {
	if len(statuses) == 0 {
		return nil, nil
	}
	var rows []taskRow
	if err := s.dbWithCtx(ctx).Where("status IN ?", statuses).Order("updated_at").Find(&rows).Error; err != nil {
		return nil, err
	}
	return tasksFromRows(rows), nil
}

func (s *Store) GetReportByRunID(ctx context.Context, runID string) (*contract.Report, error) {
	var r reportRow
	if err := s.dbWithCtx(ctx).First(&r, "run_id = ?", runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("report not found")
		}
		return nil, err
	}
	rep := reportFromRow(r)
	return &rep, nil
}

func (s *Store) GetDispatchByRound(ctx context.Context, taskID string, round int) (*contract.Dispatch, error) {
	var r dispatchRow
	if err := s.dbWithCtx(ctx).Where("task_id = ? AND round_num = ?", taskID, round).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("dispatch not found")
		}
		return nil, err
	}
	d := dispatchFromRow(r)
	return &d, nil
}

func (s *Store) GetRunByDispatchID(ctx context.Context, dispatchID string) (*contract.WorkerRun, error) {
	var r workerRunRow
	if err := s.dbWithCtx(ctx).First(&r, "dispatch_id = ?", dispatchID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("run not found")
		}
		return nil, err
	}
	run := runFromRow(r)
	return &run, nil
}

func (s *Store) LastUserMessage(ctx context.Context, taskID string) (string, error) {
	var r messageRow
	err := s.dbWithCtx(ctx).
		Where("task_id = ? AND role = ?", taskID, contract.MessageRoleUser).
		Order("created_at DESC").
		First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", errs.NotFound("message not found")
	}
	if err != nil {
		return "", err
	}
	return r.Content, nil
}

func tasksFromRows(rows []taskRow) []contract.TeamTask {
	out := make([]contract.TeamTask, len(rows))
	for i, r := range rows {
		out[i] = taskFromRow(r)
	}
	return out
}

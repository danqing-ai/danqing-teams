package sqlstore

import (
	"context"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/id"
	"gorm.io/gorm"
)

func (s *Store) ListArtifacts(ctx context.Context, teamID string) ([]contract.WorkspaceArtifact, error) {
	var rows []artifactRow
	if err := s.dbWithCtx(ctx).Where("team_id = ?", teamID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]contract.WorkspaceArtifact, len(rows))
	for i, r := range rows {
		out[i] = contract.WorkspaceArtifact{
			ID: r.ID, TeamID: r.TeamID, TaskID: r.TaskID,
			Title: r.Title, Kind: r.Kind, Content: r.Content, CreatedAt: r.CreatedAt,
		}
	}
	return out, nil
}

func (s *Store) CreateArtifact(ctx context.Context, teamID string, a contract.WorkspaceArtifact) (*contract.WorkspaceArtifact, error) {
	if a.ID == "" {
		a.ID = id.New()
	}
	a.TeamID = teamID
	a.CreatedAt = nowUTC()
	if err := s.dbWithCtx(ctx).Create(&artifactRow{
		ID: a.ID, TeamID: teamID, TaskID: a.TaskID,
		Title: a.Title, Kind: a.Kind, Content: a.Content, CreatedAt: a.CreatedAt,
	}).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) ListKnowledgeDocs(ctx context.Context, teamID, workerID string) ([]contract.KnowledgeDoc, error) {
	var rows []knowledgeDocRow
	if err := s.dbWithCtx(ctx).Where("team_id = ? AND worker_id = ?", teamID, workerID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]contract.KnowledgeDoc, len(rows))
	for i, r := range rows {
		out[i] = contract.KnowledgeDoc{ID: r.ID, Title: r.Title, Size: r.Size}
	}
	return out, nil
}

func (s *Store) SaveKnowledgeDocs(ctx context.Context, teamID, workerID string, docs []contract.KnowledgeDoc) error {
	return s.dbWithCtx(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("team_id = ? AND worker_id = ?", teamID, workerID).Delete(&knowledgeDocRow{}).Error; err != nil {
			return err
		}
		for _, d := range docs {
			if err := tx.Create(&knowledgeDocRow{
				TeamID: teamID, WorkerID: workerID, ID: d.ID, Title: d.Title, Size: d.Size,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

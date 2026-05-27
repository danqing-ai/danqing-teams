package service

import (
	"context"
	"time"

	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/domain/repository"
	"danqing-teams/pkg/id"
)

type WorkspaceService struct {
	store repository.WorkspaceRepository
}

func NewWorkspaceService(store repository.WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{store: store}
}

func (s *WorkspaceService) ListArtifacts(ctx context.Context, teamID string) ([]model.WorkspaceArtifact, error) {
	return s.store.ListArtifacts(ctx, teamID)
}

func (s *WorkspaceService) CreateArtifact(ctx context.Context, teamID string, req model.CreateArtifactRequest) (*model.WorkspaceArtifact, error) {
	a := model.WorkspaceArtifact{
		ID: id.New(), Title: req.Title, Kind: req.Kind, Content: req.Content, TaskID: req.TaskID,
		CreatedAt: time.Now().UTC(),
	}
	return s.store.CreateArtifact(ctx, teamID, a)
}

func (s *WorkspaceService) ListKnowledgeDocs(ctx context.Context, teamID, workerID string) ([]model.KnowledgeDoc, error) {
	return s.store.ListKnowledgeDocs(ctx, teamID, workerID)
}

func (s *WorkspaceService) SaveKnowledgeDocs(ctx context.Context, teamID, workerID string, req model.UpsertKnowledgeDocsRequest) error {
	return s.store.SaveKnowledgeDocs(ctx, teamID, workerID, req.Docs)
}

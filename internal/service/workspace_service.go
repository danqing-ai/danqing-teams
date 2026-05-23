package service

import (
	"context"
	"time"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/id"
)

type WorkspaceService struct {
	store contract.WorkspaceRepository
}

func NewWorkspaceService(store contract.WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{store: store}
}

func (s *WorkspaceService) ListArtifacts(ctx context.Context, teamID string) ([]contract.WorkspaceArtifact, error) {
	return s.store.ListArtifacts(ctx, teamID)
}

func (s *WorkspaceService) CreateArtifact(ctx context.Context, teamID string, req contract.CreateArtifactRequest) (*contract.WorkspaceArtifact, error) {
	a := contract.WorkspaceArtifact{
		ID: id.New(), Title: req.Title, Kind: req.Kind, Content: req.Content, TaskID: req.TaskID,
		CreatedAt: time.Now().UTC(),
	}
	return s.store.CreateArtifact(ctx, teamID, a)
}

func (s *WorkspaceService) ListKnowledgeDocs(ctx context.Context, teamID, workerID string) ([]contract.KnowledgeDoc, error) {
	return s.store.ListKnowledgeDocs(ctx, teamID, workerID)
}

func (s *WorkspaceService) SaveKnowledgeDocs(ctx context.Context, teamID, workerID string, req contract.UpsertKnowledgeDocsRequest) error {
	return s.store.SaveKnowledgeDocs(ctx, teamID, workerID, req.Docs)
}

package service

import (
	"context"

	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/domain/repository"
	"danqing-teams/pkg/id"
)

type TeamService struct {
	store repository.TeamRepository
}

func NewTeamService(store repository.TeamRepository) *TeamService {
	return &TeamService{store: store}
}

func (s *TeamService) List(ctx context.Context) ([]model.Team, error) {
	return s.store.ListTeams(ctx)
}

func (s *TeamService) Get(ctx context.Context, teamID string, controllerView bool) (*model.TeamDetail, error) {
	detail, err := s.store.GetTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if controllerView {
		personas, _ := s.store.ListPersonaCatalog(ctx, teamID)
		detail.Personas = personas
		detail.Workers = nil
	}
	return detail, nil
}

func (s *TeamService) Create(ctx context.Context, req model.CreateTeamRequest) (*model.TeamDetail, error) {
	return s.store.CreateTeam(ctx, req)
}

func (s *TeamService) Update(ctx context.Context, teamID string, req model.UpdateTeamRequest) (*model.Team, error) {
	return s.store.UpdateTeam(ctx, teamID, req)
}

func (s *TeamService) Delete(ctx context.Context, teamID string) error {
	return s.store.DeleteTeam(ctx, teamID)
}

func (s *TeamService) ListWorkers(ctx context.Context, teamID string, controllerView bool) (any, error) {
	if controllerView {
		return s.store.ListPersonaCatalog(ctx, teamID)
	}
	return s.store.ListWorkers(ctx, teamID)
}

func (s *TeamService) UpsertWorker(ctx context.Context, teamID string, req model.UpsertWorkerRequest, workerID string) (*model.WorkerAgent, error) {
	if workerID == "" {
		workerID = id.New()
	}
	w := model.WorkerAgent{
		ID: workerID, Name: req.Name, Persona: req.Persona,
		Skills: req.Skills, Tools: req.Tools, KnowledgeBase: req.KnowledgeBase,
	}
	if err := s.store.UpsertWorker(ctx, teamID, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *TeamService) DeleteWorker(ctx context.Context, teamID, workerID string) error {
	return s.store.DeleteWorker(ctx, teamID, workerID)
}

func (s *TeamService) GetController(ctx context.Context, teamID string) (*model.TeamController, error) {
	return s.store.GetController(ctx, teamID)
}

func (s *TeamService) UpdateController(ctx context.Context, teamID string, c model.TeamController) error {
	return s.store.UpdateController(ctx, teamID, c)
}

func (s *TeamService) ListHumans(ctx context.Context, teamID string) ([]model.HumanMember, error) {
	return s.store.ListHumans(ctx, teamID)
}

func (s *TeamService) AddHuman(ctx context.Context, teamID string, h model.HumanMember) error {
	return s.store.AddHuman(ctx, teamID, h)
}

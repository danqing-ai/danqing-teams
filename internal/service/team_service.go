package service

import (
	"context"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/id"
)

type TeamService struct {
	store contract.TeamRepository
}

func NewTeamService(store contract.TeamRepository) *TeamService {
	return &TeamService{store: store}
}

func (s *TeamService) List(ctx context.Context) ([]contract.Team, error) {
	return s.store.ListTeams(ctx)
}

func (s *TeamService) Get(ctx context.Context, teamID string, controllerView bool) (*contract.TeamDetail, error) {
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

func (s *TeamService) Create(ctx context.Context, req contract.CreateTeamRequest) (*contract.TeamDetail, error) {
	return s.store.CreateTeam(ctx, req)
}

func (s *TeamService) Update(ctx context.Context, teamID string, req contract.UpdateTeamRequest) (*contract.Team, error) {
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

func (s *TeamService) UpsertWorker(ctx context.Context, teamID string, req contract.UpsertWorkerRequest, workerID string) (*contract.WorkerAgent, error) {
	if workerID == "" {
		workerID = id.New()
	}
	w := contract.WorkerAgent{
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

func (s *TeamService) GetController(ctx context.Context, teamID string) (*contract.TeamController, error) {
	return s.store.GetController(ctx, teamID)
}

func (s *TeamService) UpdateController(ctx context.Context, teamID string, c contract.TeamController) error {
	return s.store.UpdateController(ctx, teamID, c)
}

func (s *TeamService) ListHumans(ctx context.Context, teamID string) ([]contract.HumanMember, error) {
	return s.store.ListHumans(ctx, teamID)
}

func (s *TeamService) AddHuman(ctx context.Context, teamID string, h contract.HumanMember) error {
	return s.store.AddHuman(ctx, teamID, h)
}

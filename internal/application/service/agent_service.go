package service

import (
	"context"

	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/domain/repository"
)

type AgentService struct {
	store repository.AgentRepository
	teams repository.TeamRepository
}

func NewAgentService(store repository.AgentRepository, teams repository.TeamRepository) *AgentService {
	return &AgentService{store: store, teams: teams}
}

func (s *AgentService) List(ctx context.Context, role model.AgentRole) ([]model.Agent, error) {
	return s.store.ListAgents(ctx, role)
}

func (s *AgentService) Get(ctx context.Context, agentID string) (*model.Agent, error) {
	return s.store.GetAgent(ctx, agentID)
}

func (s *AgentService) Create(ctx context.Context, req model.CreateAgentRequest) (*model.Agent, error) {
	return s.store.CreateAgent(ctx, req)
}

func (s *AgentService) Update(ctx context.Context, agentID string, req model.UpdateAgentRequest) (*model.Agent, error) {
	return s.store.UpdateAgent(ctx, agentID, req)
}

func (s *AgentService) Delete(ctx context.Context, agentID string) error {
	return s.store.DeleteAgent(ctx, agentID)
}

func (s *AgentService) ListTeamMembers(ctx context.Context, teamID string) ([]model.Agent, error) {
	return s.store.ListTeamAgentMembers(ctx, teamID)
}

func (s *AgentService) AddToTeam(ctx context.Context, teamID, agentID string) error {
	return s.store.AddTeamAgent(ctx, teamID, agentID)
}

func (s *AgentService) RemoveFromTeam(ctx context.Context, teamID, agentID string) error {
	return s.store.RemoveTeamAgent(ctx, teamID, agentID)
}

package service

import (
	"context"

	"danqing-teams/internal/contract"
)

type AgentService struct {
	store contract.AgentRepository
	teams contract.TeamRepository
}

func NewAgentService(store contract.AgentRepository, teams contract.TeamRepository) *AgentService {
	return &AgentService{store: store, teams: teams}
}

func (s *AgentService) List(ctx context.Context, role contract.AgentRole) ([]contract.Agent, error) {
	return s.store.ListAgents(ctx, role)
}

func (s *AgentService) Get(ctx context.Context, agentID string) (*contract.Agent, error) {
	return s.store.GetAgent(ctx, agentID)
}

func (s *AgentService) Create(ctx context.Context, req contract.CreateAgentRequest) (*contract.Agent, error) {
	return s.store.CreateAgent(ctx, req)
}

func (s *AgentService) Update(ctx context.Context, agentID string, req contract.UpdateAgentRequest) (*contract.Agent, error) {
	return s.store.UpdateAgent(ctx, agentID, req)
}

func (s *AgentService) Delete(ctx context.Context, agentID string) error {
	return s.store.DeleteAgent(ctx, agentID)
}

func (s *AgentService) ListTeamMembers(ctx context.Context, teamID string) ([]contract.Agent, error) {
	return s.store.ListTeamAgentMembers(ctx, teamID)
}

func (s *AgentService) AddToTeam(ctx context.Context, teamID, agentID string) error {
	return s.store.AddTeamAgent(ctx, teamID, agentID)
}

func (s *AgentService) RemoveFromTeam(ctx context.Context, teamID, agentID string) error {
	return s.store.RemoveTeamAgent(ctx, teamID, agentID)
}

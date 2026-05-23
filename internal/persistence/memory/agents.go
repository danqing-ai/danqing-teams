package memory

import (
	"context"
	"strings"
	"time"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/errs"
)

func agentToWorker(a contract.Agent) contract.WorkerAgent {
	return contract.WorkerAgent{
		ID:            a.ID,
		Name:          a.Name,
		Persona:       a.Description,
		Skills:        a.Skills,
		Tools:         a.Tools,
		KnowledgeBase: a.KnowledgeBase,
	}
}

func sanitizeAgentForResponse(a *contract.Agent) {
	if a.LLM.APIKey != "" {
		a.LLM.HasAPIKey = true
		a.LLM.APIKey = ""
	}
}

func (s *Store) ListAgents(_ context.Context, role contract.AgentRole) ([]contract.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []contract.Agent
	for _, a := range s.agents {
		if role != "" && a.Role != role {
			continue
		}
		copy := a
		sanitizeAgentForResponse(&copy)
		out = append(out, copy)
	}
	return out, nil
}

func (s *Store) GetAgent(_ context.Context, agentID string) (*contract.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.agents[agentID]
	if !ok {
		return nil, errs.NotFound("agent not found")
	}
	copy := a
	sanitizeAgentForResponse(&copy)
	return &copy, nil
}

func (s *Store) CreateAgent(_ context.Context, req contract.CreateAgentRequest) (*contract.Agent, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errs.BadRequest("agent name required")
	}
	agentID, err := contract.ParseAgentID(req.ID)
	if err != nil {
		return nil, err
	}
	if req.Role == "" {
		req.Role = contract.AgentRoleTeamWorker
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.agents[agentID]; ok {
		return nil, errs.BadRequest("agent id already exists")
	}
	now := time.Now().UTC()
	a := contract.Agent{
		ID:                       agentID,
		Name:                     strings.TrimSpace(req.Name),
		Description:              req.Description,
		Role:                     req.Role,
		LLM:                      req.LLM,
		SystemPrompt:             req.SystemPrompt,
		MinFunctionCallingRounds: req.MinFunctionCallingRounds,
		Skills:                   req.Skills,
		Tools:                    req.Tools,
		KnowledgeBase:            req.KnowledgeBase,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if a.MinFunctionCallingRounds <= 0 {
		a.MinFunctionCallingRounds = 1
	}
	s.agents[a.ID] = a
	copy := a
	sanitizeAgentForResponse(&copy)
	return &copy, nil
}

func (s *Store) UpdateAgent(_ context.Context, agentID string, req contract.UpdateAgentRequest) (*contract.Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[agentID]
	if !ok {
		return nil, errs.NotFound("agent not found")
	}
	if req.Name != nil {
		a.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		a.Description = *req.Description
	}
	if req.Role != nil {
		a.Role = *req.Role
	}
	if req.LLM != nil {
		if req.LLM.URL != "" || req.LLM.DefaultModel != "" || len(req.LLM.AllModels) > 0 {
			a.LLM.URL = req.LLM.URL
			a.LLM.DefaultModel = req.LLM.DefaultModel
			a.LLM.AllModels = req.LLM.AllModels
		}
		if req.LLM.APIKey != "" {
			a.LLM.APIKey = req.LLM.APIKey
		}
	}
	if req.SystemPrompt != nil {
		a.SystemPrompt = *req.SystemPrompt
	}
	if req.MinFunctionCallingRounds != nil {
		a.MinFunctionCallingRounds = *req.MinFunctionCallingRounds
	}
	if req.Skills != nil {
		a.Skills = *req.Skills
	}
	if req.Tools != nil {
		a.Tools = *req.Tools
	}
	if req.KnowledgeBase != nil {
		a.KnowledgeBase = *req.KnowledgeBase
	}
	a.UpdatedAt = time.Now().UTC()
	s.agents[agentID] = a
	copy := a
	sanitizeAgentForResponse(&copy)
	return &copy, nil
}

func (s *Store) DeleteAgent(_ context.Context, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.agents[agentID]; !ok {
		return errs.NotFound("agent not found")
	}
	delete(s.agents, agentID)
	for key := range s.teamAgents {
		if strings.HasSuffix(key, ":"+agentID) {
			delete(s.teamAgents, key)
		}
	}
	return nil
}

func teamAgentKey(teamID, agentID string) string {
	return teamID + ":" + agentID
}

func (s *Store) ListTeamAgentMembers(_ context.Context, teamID string) ([]contract.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.teams[teamID]; !ok {
		return nil, errs.NotFound("team not found")
	}
	var out []contract.Agent
	for key := range s.teamAgents {
		if !strings.HasPrefix(key, teamID+":") {
			continue
		}
		agentID := strings.TrimPrefix(key, teamID+":")
		a, ok := s.agents[agentID]
		if !ok || a.Role != contract.AgentRoleTeamWorker {
			continue
		}
		copy := a
		sanitizeAgentForResponse(&copy)
		out = append(out, copy)
	}
	return out, nil
}

func (s *Store) AddTeamAgent(_ context.Context, teamID, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.teams[teamID]; !ok {
		return errs.NotFound("team not found")
	}
	a, ok := s.agents[agentID]
	if !ok {
		return errs.NotFound("agent not found")
	}
	if a.Role != contract.AgentRoleTeamWorker {
		return errs.BadRequest("only team-worker agents can join a team")
	}
	s.teamAgents[teamAgentKey(teamID, agentID)] = struct{}{}
	return nil
}

func (s *Store) RemoveTeamAgent(_ context.Context, teamID, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := teamAgentKey(teamID, agentID)
	if _, ok := s.teamAgents[key]; !ok {
		return errs.NotFound("agent not in team")
	}
	delete(s.teamAgents, key)
	return nil
}

func (s *Store) IsTeamAgentMember(_ context.Context, teamID, agentID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.teamAgents[teamAgentKey(teamID, agentID)]
	return ok, nil
}

func (s *Store) listWorkersFromAgents(teamID string) []contract.WorkerAgent {
	var out []contract.WorkerAgent
	for key := range s.teamAgents {
		if !strings.HasPrefix(key, teamID+":") {
			continue
		}
		agentID := strings.TrimPrefix(key, teamID+":")
		a, ok := s.agents[agentID]
		if !ok || a.Role != contract.AgentRoleTeamWorker {
			continue
		}
		out = append(out, agentToWorker(a))
	}
	return out
}

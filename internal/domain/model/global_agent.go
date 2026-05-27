package model

import (
	"regexp"
	"strings"
	"time"

	"danqing-teams/pkg/errs"
)

var agentIDPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{1,63}$`)

func ParseAgentID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errs.BadRequest("agent id required")
	}
	if !agentIDPattern.MatchString(id) {
		return "", errs.BadRequest("agent id must start with a letter and use letters, digits, _ or -")
	}
	return id, nil
}

type AgentRole string

const (
	AgentRoleTeamWorker     AgentRole = "team-worker"
	AgentRoleTeamController AgentRole = "team-controller"
)

type AgentLLMConfig struct {
	URL          string
	APIKey       string
	HasAPIKey    bool
	DefaultModel string
	AllModels    []string
}

type Agent struct {
	ID                       string
	Name                     string
	Description              string
	Role                     AgentRole
	LLM                      AgentLLMConfig
	SystemPrompt             string
	MinFunctionCallingRounds int
	Skills                   []Skill
	Tools                    []ToolBinding
	KnowledgeBase            KnowledgeBaseRef
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type CreateAgentRequest struct {
	ID                       string
	Name                     string
	Description              string
	Role                     AgentRole
	LLM                      AgentLLMConfig
	SystemPrompt             string
	MinFunctionCallingRounds int
	Skills                   []Skill
	Tools                    []ToolBinding
	KnowledgeBase            KnowledgeBaseRef
}

type UpdateAgentRequest struct {
	Name                     *string
	Description              *string
	Role                     *AgentRole
	LLM                      *AgentLLMConfig
	SystemPrompt             *string
	MinFunctionCallingRounds *int
	Skills                   *[]Skill
	Tools                    *[]ToolBinding
	KnowledgeBase            *KnowledgeBaseRef
}

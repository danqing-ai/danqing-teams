package dto

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
	URL          string   `json:"url,omitempty"`
	APIKey       string   `json:"apiKey,omitempty"`
	HasAPIKey    bool     `json:"hasApiKey,omitempty"`
	DefaultModel string   `json:"defaultModel,omitempty"`
	AllModels    []string `json:"allModels,omitempty"`
}

type Agent struct {
	ID                       string           `json:"id"`
	Name                     string           `json:"name"`
	Description              string           `json:"description"`
	Role                     AgentRole        `json:"role"`
	LLM                      AgentLLMConfig   `json:"llm"`
	SystemPrompt             string           `json:"systemPrompt,omitempty"`
	MinFunctionCallingRounds int              `json:"minFunctionCallingRounds"`
	Skills                   []Skill          `json:"skills,omitempty"`
	Tools                    []ToolBinding    `json:"tools,omitempty"`
	KnowledgeBase            KnowledgeBaseRef `json:"knowledgeBase,omitempty"`
	CreatedAt                time.Time        `json:"createdAt,omitempty"`
	UpdatedAt                time.Time        `json:"updatedAt,omitempty"`
}

type CreateAgentRequest struct {
	ID                       string           `json:"id"`
	Name                     string           `json:"name"`
	Description              string           `json:"description"`
	Role                     AgentRole        `json:"role"`
	LLM                      AgentLLMConfig   `json:"llm"`
	SystemPrompt             string           `json:"systemPrompt"`
	MinFunctionCallingRounds int              `json:"minFunctionCallingRounds"`
	Skills                   []Skill          `json:"skills,omitempty"`
	Tools                    []ToolBinding    `json:"tools,omitempty"`
	KnowledgeBase            KnowledgeBaseRef `json:"knowledgeBase,omitempty"`
}

type UpdateAgentRequest struct {
	Name                     *string           `json:"name,omitempty"`
	Description              *string           `json:"description,omitempty"`
	Role                     *AgentRole        `json:"role,omitempty"`
	LLM                      *AgentLLMConfig   `json:"llm,omitempty"`
	SystemPrompt             *string           `json:"systemPrompt,omitempty"`
	MinFunctionCallingRounds *int              `json:"minFunctionCallingRounds,omitempty"`
	Skills                   *[]Skill          `json:"skills,omitempty"`
	Tools                    *[]ToolBinding    `json:"tools,omitempty"`
	KnowledgeBase            *KnowledgeBaseRef `json:"knowledgeBase,omitempty"`
}

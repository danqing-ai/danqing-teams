package port

import (
	"context"
	"danqing-teams/core/domain"
)

type Repository interface {
	Agents()       AgentRepo
	Skills()       SkillRepo
	SkillFiles()   SkillFileRepo
	Sessions()     SessionRepo
	Projects()     ProjectRepo
	LLMConfig()    LLMConfigRepo
	Approvals()    ApprovalRepo
	StreamEvents() StreamEventRepo
	Turns()        TurnRepo
	MCPServers()   MCPServerRepo
}

type AgentRepo interface {
	List(ctx context.Context) ([]domain.Agent, error)
	Get(ctx context.Context, id string) (domain.Agent, error)
	Upsert(ctx context.Context, a domain.Agent) error
	Delete(ctx context.Context, id string) error
}

type SkillRepo interface {
	List(ctx context.Context) ([]domain.Skill, error)
	Get(ctx context.Context, id string) (domain.Skill, error)
	Upsert(ctx context.Context, s domain.Skill) error
	Delete(ctx context.Context, id string) error
}

type SkillFileRepo interface {
	ListBySkill(ctx context.Context, skillID string) ([]domain.SkillFile, error)
	Get(ctx context.Context, skillID, path string) (domain.SkillFile, error)
	Upsert(ctx context.Context, f domain.SkillFile) error
	Delete(ctx context.Context, skillID, path string) error
	DeleteBySkill(ctx context.Context, skillID string) error
}

type SessionRepo interface {
	Create(ctx context.Context, s domain.Session) error
	Update(ctx context.Context, s domain.Session) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (domain.Session, error)
	List(ctx context.Context) ([]domain.Session, error)
	ListByProject(ctx context.Context, projectID string) ([]domain.Session, error)
}

type ProjectRepo interface {
	Create(ctx context.Context, p domain.Project) error
	Update(ctx context.Context, p domain.Project) error
	Get(ctx context.Context, id string) (domain.Project, error)
	List(ctx context.Context) ([]domain.Project, error)
	Delete(ctx context.Context, id string) error
}

type LLMConfigRepo interface {
	GetAll(ctx context.Context) ([]domain.LLMProviderConfig, error)
	GetByID(ctx context.Context, id string) (domain.LLMProviderConfig, error)
	Upsert(ctx context.Context, cfg domain.LLMProviderConfig) error
	Delete(ctx context.Context, id string) error
}

type SearchConfigStore interface {
	Get(ctx context.Context) (domain.SearchConfig, error)
	Upsert(ctx context.Context, cfg domain.SearchConfig) error
}

type ConfigStore interface {
	Load(ctx context.Context) (*domain.ConfigFile, error)
	Save(ctx context.Context, cfg *domain.ConfigFile) error
}

type ApprovalRepo interface {
	Create(ctx context.Context, a domain.Approval) error
	Get(ctx context.Context, id string) (domain.Approval, error)
	Update(ctx context.Context, a domain.Approval) error
	ListByStatus(ctx context.Context, status string) ([]domain.Approval, error)
}

type StreamEventRepo interface {
	Save(ctx context.Context, event domain.StreamEvent) error
	ListBySession(ctx context.Context, sessionID string, since int64) ([]domain.StreamEvent, error)
	MaxSeq() int64
}

type TurnRepo interface {
	Create(ctx context.Context, t domain.TurnLog) error
	UpdateStatus(ctx context.Context, id string, status domain.TurnStatus) error
	Get(ctx context.Context, id string) (domain.TurnLog, error)
	ListBySession(ctx context.Context, sessionID string) ([]domain.TurnLog, error)
	ListByStatus(ctx context.Context, status domain.TurnStatus) ([]domain.TurnLog, error)
}

type MCPServerRepo interface {
	List(ctx context.Context) ([]domain.MCPServer, error)
	Get(ctx context.Context, id string) (domain.MCPServer, error)
	Upsert(ctx context.Context, s domain.MCPServer) error
	Delete(ctx context.Context, id string) error
}

// TurnLogStore persists turn-level JSONL entries used exclusively for LLM
// message reconstruction (turn recovery) and offline debugging (zip download).
//
// WHITELIST of allowed entry types:
//   - "start"        — written by Create (skipped on reopen/resume of existing file)
//   - "tool_call"    — written by Append before tool Execute
//   - "tool_result"  — written by Append after tool Execute (success or error)
//   - "end"          — written by EndTurn
//
// DO NOT write diagnostic, audit, or telemetry entries here (e.g. llm_error,
// step events, permission decisions). Those belong in Stream Events
// (port.EventStream) which serve the UI/SSE timeline.
//
// LoadForRecovery enforces this whitelist: only tool_call / tool_result
// entries participate in message reconstruction; all others are skipped.
// An unpaired trailing tool_call is dropped so earlier complete pairs survive.
type TurnLogStore interface {
	Create(turnID, sessionID, projectID, agentID, goal string) error
	Append(turnID, typ string, data map[string]any)
	EndTurn(turnID string, status domain.TurnStatus)
	LastStatus(sessionID string) domain.TurnStatus
	ListTurns(sessionID string) []domain.TurnLog
	LoadForRecovery(turnID string) (goal string, entries []map[string]any)
	LoadRawLog(turnID string) ([]byte, error)
	LoadTurnLogZip(turnID string, events []domain.StreamEvent) ([]byte, error)
}

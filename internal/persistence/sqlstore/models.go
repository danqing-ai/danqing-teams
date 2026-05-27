package sqlstore

import (
	"time"

	"danqing-teams/internal/domain/model"
)

type teamRow struct {
	ID          string `gorm:"primaryKey"`
	Name        string
	Description string `gorm:"default:''"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (teamRow) TableName() string { return "teams" }

type teamControllerRow struct {
	TeamID       string `gorm:"primaryKey"`
	Persona      string `gorm:"default:''"`
	SystemPrompt string `gorm:"column:system_prompt;default:''"`
}

func (teamControllerRow) TableName() string { return "team_controllers" }

type workerRow struct {
	ID       string `gorm:"primaryKey"`
	TeamID   string `gorm:"index:idx_workers_team"`
	Name     string
	Persona  string                    `gorm:"default:''"`
	Skills   []model.Skill          `gorm:"column:skills_json;serializer:json;default:'[]'"`
	Tools    []model.ToolBinding    `gorm:"column:tools_json;serializer:json;default:'[]'"`
	KB       model.KnowledgeBaseRef `gorm:"column:kb_json;serializer:json"`
}

func (workerRow) TableName() string { return "workers" }

type humanRow struct {
	ID          string `gorm:"primaryKey"`
	TeamID      string `gorm:"index:idx_humans_team"`
	DisplayName string `gorm:"column:display_name"`
	Email       string `gorm:"default:''"`
	Role        string `gorm:"default:observer"`
}

func (humanRow) TableName() string { return "humans" }

type taskRow struct {
	ID          string                  `gorm:"primaryKey"`
	TeamID      string                  `gorm:"index:idx_tasks_team"`
	Content     string
	Status      model.TaskStatus     `gorm:"index:idx_tasks_status"`
	CloseReason model.TaskCloseReason `gorm:"column:close_reason;default:''"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (taskRow) TableName() string { return "tasks" }

type dispatchRow struct {
	ID             string `gorm:"primaryKey"`
	TaskID         string `gorm:"index:idx_dispatches_task"`
	WorkerID       string `gorm:"column:worker_id"`
	WorkerName     string `gorm:"column:worker_name;default:''"`
	Intent         string
	ContextSummary string `gorm:"column:context_summary;default:''"`
	Round          int    `gorm:"column:round_num;default:0"`
	CreatedAt      time.Time
}

func (dispatchRow) TableName() string { return "dispatches" }

type workerRunRow struct {
	ID         string              `gorm:"primaryKey"`
	TaskID     string              `gorm:"index:idx_runs_task"`
	DispatchID string              `gorm:"column:dispatch_id"`
	WorkerID   string              `gorm:"column:worker_id"`
	Status     model.RunStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (workerRunRow) TableName() string { return "worker_runs" }

type executionPlanRow struct {
	RunID         string              `gorm:"primaryKey;column:run_id"`
	SkillIDs      []string            `gorm:"column:skill_ids_json;serializer:json;default:'[]'"`
	ToolIDs       []string            `gorm:"column:tool_ids_json;serializer:json;default:'[]'"`
	Rationale     string              `gorm:"default:''"`
	EvaluatedRisk model.RiskLevel  `gorm:"column:evaluated_risk;default:low"`
	HighRiskItems []model.RiskItem `gorm:"column:high_risk_json;serializer:json;default:'[]'"`
}

func (executionPlanRow) TableName() string { return "execution_plans" }

type reportRow struct {
	ID               string                    `gorm:"primaryKey"`
	RunID            string                    `gorm:"uniqueIndex:idx_reports_run;column:run_id"`
	TaskID           string                    `gorm:"index:idx_reports_task"`
	WorkerID         string                    `gorm:"column:worker_id"`
	WorkerName       string                    `gorm:"column:worker_name;default:''"`
	ContentMarkdown  string                    `gorm:"column:content_markdown"`
	Intent           model.ReportIntent
	SuggestedActions []model.SuggestedAction `gorm:"column:suggested_actions_json;serializer:json;default:'[]'"`
	CreatedAt        time.Time
}

func (reportRow) TableName() string { return "reports" }

type timelineEventRow struct {
	ID        string `gorm:"primaryKey"`
	TaskID    string `gorm:"index:idx_timeline_task"`
	Type      string
	Payload   any `gorm:"column:payload_json;serializer:json"`
	CreatedAt time.Time
}

func (timelineEventRow) TableName() string { return "timeline_events" }

type messageRow struct {
	ID        string               `gorm:"primaryKey"`
	TeamID    string               `gorm:"column:team_id"`
	TaskID    string               `gorm:"index:idx_messages_task"`
	Role      model.MessageRole
	Content   string
	CreatedAt time.Time
}

func (messageRow) TableName() string { return "messages" }

type approvalRow struct {
	ID            string                  `gorm:"primaryKey"`
	TeamID        string                  `gorm:"index:idx_approvals_team"`
	TaskID        string                  `gorm:"column:task_id"`
	RunID         string                  `gorm:"index:idx_approvals_run;column:run_id"`
	Summary       string
	HighRiskItems []model.RiskItem     `gorm:"column:high_risk_json;serializer:json;default:'[]'"`
	Status        model.ApprovalStatus
	Comment       string                  `gorm:"default:''"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (approvalRow) TableName() string { return "approvals" }

type todoRow struct {
	ID        string `gorm:"primaryKey"`
	TeamID    string `gorm:"index:idx_todos_team"`
	TaskID    string `gorm:"column:task_id;default:''"`
	Title     string
	Done      bool `gorm:"default:false"`
	CreatedAt time.Time
}

func (todoRow) TableName() string { return "todos" }

type artifactRow struct {
	ID        string `gorm:"primaryKey"`
	TeamID    string `gorm:"index:idx_artifacts_team"`
	TaskID    string `gorm:"column:task_id;default:''"`
	Title     string
	Kind      string
	Content   string `gorm:"default:''"`
	CreatedAt time.Time
}

func (artifactRow) TableName() string { return "artifacts" }

type knowledgeDocRow struct {
	TeamID   string `gorm:"primaryKey"`
	WorkerID string `gorm:"primaryKey;column:worker_id"`
	ID       string `gorm:"primaryKey"`
	Title    string
	Size     int `gorm:"default:0"`
}

func (knowledgeDocRow) TableName() string { return "knowledge_docs" }

type orchestrationJobRow struct {
	ID         string             `gorm:"primaryKey"`
	TeamID     string             `gorm:"column:team_id"`
	TaskID     string             `gorm:"index:idx_jobs_task;column:task_id"`
	Kind       model.JobKind
	Payload    string             `gorm:"column:payload_json;default:'{}'"`
	DedupKey   string             `gorm:"index:idx_jobs_dedup;column:dedup_key"`
	Status     model.JobStatus `gorm:"index:idx_jobs_status_created,priority:1"`
	LeaseOwner string             `gorm:"column:lease_owner;default:''"`
	LeaseUntil *time.Time         `gorm:"column:lease_until"`
	LastError  string             `gorm:"column:last_error;default:''"`
	CreatedAt  time.Time          `gorm:"index:idx_jobs_status_created,priority:2"`
	UpdatedAt  time.Time
}

func (orchestrationJobRow) TableName() string { return "orchestration_jobs" }

type agentRow struct {
	ID                       string               `gorm:"primaryKey"`
	Name                     string
	Description              string               `gorm:"default:''"`
	Role                     model.AgentRole   `gorm:"index:idx_agents_role"`
	LLMURL                   string               `gorm:"column:llm_url;default:''"`
	LLMAPIKey                string               `gorm:"column:llm_api_key;default:''"`
	DefaultModel             string               `gorm:"column:default_model;default:''"`
	AllModels                []string             `gorm:"column:all_models_json;serializer:json;default:'[]'"`
	SystemPrompt             string               `gorm:"column:system_prompt;default:''"`
	MinFunctionCallingRounds int                  `gorm:"column:min_function_calling_rounds;default:1"`
	Skills                   []model.Skill     `gorm:"column:skills_json;serializer:json;default:'[]'"`
	Tools                    []model.ToolBinding `gorm:"column:tools_json;serializer:json;default:'[]'"`
	KB                       model.KnowledgeBaseRef `gorm:"column:kb_json;serializer:json"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (agentRow) TableName() string { return "agents" }

type teamAgentRow struct {
	TeamID  string `gorm:"primaryKey"`
	AgentID string `gorm:"primaryKey;index:idx_team_agents_agent"`
}

func (teamAgentRow) TableName() string { return "team_agents" }

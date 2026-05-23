package contract

import "context"

type TeamRepository interface {
	ListTeams(ctx context.Context) ([]Team, error)
	GetTeam(ctx context.Context, teamID string) (*TeamDetail, error)
	CreateTeam(ctx context.Context, req CreateTeamRequest) (*TeamDetail, error)
	UpdateTeam(ctx context.Context, teamID string, req UpdateTeamRequest) (*Team, error)
	DeleteTeam(ctx context.Context, teamID string) error

	ListPersonaCatalog(ctx context.Context, teamID string) ([]WorkerPersonaCatalog, error)
	GetController(ctx context.Context, teamID string) (*TeamController, error)
	UpdateController(ctx context.Context, teamID string, c TeamController) error

	ListWorkers(ctx context.Context, teamID string) ([]WorkerAgent, error)
	GetWorker(ctx context.Context, teamID, workerID string) (*WorkerAgent, error)
	UpsertWorker(ctx context.Context, teamID string, worker *WorkerAgent) error
	DeleteWorker(ctx context.Context, teamID, workerID string) error
	GetWorkerPrivateProfile(ctx context.Context, teamID, workerID string) (*WorkerPrivateProfile, error)

	ListHumans(ctx context.Context, teamID string) ([]HumanMember, error)
	AddHuman(ctx context.Context, teamID string, h HumanMember) error

	ListTodos(ctx context.Context, teamID string, taskID string) ([]TodoItem, error)
	CreateTodo(ctx context.Context, teamID string, item TodoItem) (*TodoItem, error)
	UpdateTodo(ctx context.Context, teamID, todoID string, done bool) (*TodoItem, error)
}

type TaskRepository interface {
	ListTasks(ctx context.Context, teamID string, status TaskStatus) ([]TeamTask, error)
	GetTask(ctx context.Context, teamID, taskID string) (*TeamTask, error)
	CreateTask(ctx context.Context, task *TeamTask) error
	UpdateTaskStatus(ctx context.Context, teamID, taskID string, status TaskStatus) error
	UpdateTaskClosure(ctx context.Context, teamID, taskID string, status TaskStatus, reason TaskCloseReason) error

	SaveDispatch(ctx context.Context, d *Dispatch) error
	ListDispatches(ctx context.Context, taskID string) ([]Dispatch, error)

	SaveRun(ctx context.Context, run *WorkerRun) error
	GetRun(ctx context.Context, runID string) (*WorkerRun, error)
	UpdateRun(ctx context.Context, run *WorkerRun) error
	ListRuns(ctx context.Context, taskID string) ([]WorkerRun, error)

	SavePlan(ctx context.Context, plan *ExecutionPlan) error
	GetPlan(ctx context.Context, runID string) (*ExecutionPlan, error)

	SaveReport(ctx context.Context, r *Report) error
	ListReports(ctx context.Context, taskID string) ([]Report, error)

	AppendTimeline(ctx context.Context, evt TimelineEvent) error
	GetTimeline(ctx context.Context, taskID string) ([]TimelineEvent, error)

	AppendMessage(ctx context.Context, msg *TeamMessage) error
	ListMessages(ctx context.Context, taskID string) ([]TeamMessage, error)
}

type WorkspaceRepository interface {
	ListArtifacts(ctx context.Context, teamID string) ([]WorkspaceArtifact, error)
	CreateArtifact(ctx context.Context, teamID string, a WorkspaceArtifact) (*WorkspaceArtifact, error)
	ListKnowledgeDocs(ctx context.Context, teamID, workerID string) ([]KnowledgeDoc, error)
	SaveKnowledgeDocs(ctx context.Context, teamID, workerID string, docs []KnowledgeDoc) error
}

type ApprovalRepository interface {
	Create(ctx context.Context, req *ApprovalRequest) error
	Get(ctx context.Context, teamID, approvalID string) (*ApprovalRequest, error)
	Update(ctx context.Context, req *ApprovalRequest) error
	List(ctx context.Context, teamID string, status ApprovalStatus) ([]ApprovalRequest, error)
	GetByRunID(ctx context.Context, runID string) (*ApprovalRequest, error)
}

type AgentRepository interface {
	ListAgents(ctx context.Context, role AgentRole) ([]Agent, error)
	GetAgent(ctx context.Context, agentID string) (*Agent, error)
	CreateAgent(ctx context.Context, req CreateAgentRequest) (*Agent, error)
	UpdateAgent(ctx context.Context, agentID string, req UpdateAgentRequest) (*Agent, error)
	DeleteAgent(ctx context.Context, agentID string) error

	ListTeamAgentMembers(ctx context.Context, teamID string) ([]Agent, error)
	AddTeamAgent(ctx context.Context, teamID, agentID string) error
	RemoveTeamAgent(ctx context.Context, teamID, agentID string) error
	IsTeamAgentMember(ctx context.Context, teamID, agentID string) (bool, error)
}

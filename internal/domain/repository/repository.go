package repository

import (
	"context"
	"time"

	"danqing-teams/internal/domain/model"
)

type TeamRepository interface {
	ListTeams(ctx context.Context) ([]model.Team, error)
	GetTeam(ctx context.Context, teamID string) (*model.TeamDetail, error)
	CreateTeam(ctx context.Context, req model.CreateTeamRequest) (*model.TeamDetail, error)
	UpdateTeam(ctx context.Context, teamID string, req model.UpdateTeamRequest) (*model.Team, error)
	DeleteTeam(ctx context.Context, teamID string) error

	ListPersonaCatalog(ctx context.Context, teamID string) ([]model.WorkerPersonaCatalog, error)
	GetController(ctx context.Context, teamID string) (*model.TeamController, error)
	UpdateController(ctx context.Context, teamID string, c model.TeamController) error

	ListWorkers(ctx context.Context, teamID string) ([]model.WorkerAgent, error)
	GetWorker(ctx context.Context, teamID, workerID string) (*model.WorkerAgent, error)
	UpsertWorker(ctx context.Context, teamID string, worker *model.WorkerAgent) error
	DeleteWorker(ctx context.Context, teamID, workerID string) error
	GetWorkerPrivateProfile(ctx context.Context, teamID, workerID string) (*model.WorkerPrivateProfile, error)

	ListHumans(ctx context.Context, teamID string) ([]model.HumanMember, error)
	AddHuman(ctx context.Context, teamID string, h model.HumanMember) error

	ListTodos(ctx context.Context, teamID string, taskID string) ([]model.TodoItem, error)
	CreateTodo(ctx context.Context, teamID string, item model.TodoItem) (*model.TodoItem, error)
	UpdateTodo(ctx context.Context, teamID, todoID string, done bool) (*model.TodoItem, error)
}

type TaskRepository interface {
	ListTasks(ctx context.Context, teamID string, status model.TaskStatus) ([]model.TeamTask, error)
	GetTask(ctx context.Context, teamID, taskID string) (*model.TeamTask, error)
	CreateTask(ctx context.Context, task *model.TeamTask) error
	UpdateTaskStatus(ctx context.Context, teamID, taskID string, status model.TaskStatus) error
	UpdateTaskClosure(ctx context.Context, teamID, taskID string, status model.TaskStatus, reason model.TaskCloseReason) error

	SaveDispatch(ctx context.Context, d *model.Dispatch) error
	ListDispatches(ctx context.Context, taskID string) ([]model.Dispatch, error)

	SaveRun(ctx context.Context, run *model.WorkerRun) error
	GetRun(ctx context.Context, runID string) (*model.WorkerRun, error)
	UpdateRun(ctx context.Context, run *model.WorkerRun) error
	ListRuns(ctx context.Context, taskID string) ([]model.WorkerRun, error)

	SavePlan(ctx context.Context, plan *model.ExecutionPlan) error
	GetPlan(ctx context.Context, runID string) (*model.ExecutionPlan, error)

	SaveReport(ctx context.Context, r *model.Report) error
	ListReports(ctx context.Context, taskID string) ([]model.Report, error)

	AppendTimeline(ctx context.Context, evt model.TimelineEvent) error
	GetTimeline(ctx context.Context, taskID string) ([]model.TimelineEvent, error)

	AppendMessage(ctx context.Context, msg *model.TeamMessage) error
	ListMessages(ctx context.Context, taskID string) ([]model.TeamMessage, error)
}

type WorkspaceRepository interface {
	ListArtifacts(ctx context.Context, teamID string) ([]model.WorkspaceArtifact, error)
	CreateArtifact(ctx context.Context, teamID string, a model.WorkspaceArtifact) (*model.WorkspaceArtifact, error)
	ListKnowledgeDocs(ctx context.Context, teamID, workerID string) ([]model.KnowledgeDoc, error)
	SaveKnowledgeDocs(ctx context.Context, teamID, workerID string, docs []model.KnowledgeDoc) error
}

type ApprovalRepository interface {
	Create(ctx context.Context, req *model.ApprovalRequest) error
	Get(ctx context.Context, teamID, approvalID string) (*model.ApprovalRequest, error)
	Update(ctx context.Context, req *model.ApprovalRequest) error
	List(ctx context.Context, teamID string, status model.ApprovalStatus) ([]model.ApprovalRequest, error)
	GetByRunID(ctx context.Context, runID string) (*model.ApprovalRequest, error)
}

type AgentRepository interface {
	ListAgents(ctx context.Context, role model.AgentRole) ([]model.Agent, error)
	GetAgent(ctx context.Context, agentID string) (*model.Agent, error)
	CreateAgent(ctx context.Context, req model.CreateAgentRequest) (*model.Agent, error)
	UpdateAgent(ctx context.Context, agentID string, req model.UpdateAgentRequest) (*model.Agent, error)
	DeleteAgent(ctx context.Context, agentID string) error

	ListTeamAgentMembers(ctx context.Context, teamID string) ([]model.Agent, error)
	AddTeamAgent(ctx context.Context, teamID, agentID string) error
	RemoveTeamAgent(ctx context.Context, teamID, agentID string) error
	IsTeamAgentMember(ctx context.Context, teamID, agentID string) (bool, error)
}

type JobRepository interface {
	Enqueue(ctx context.Context, job *model.OrchestrationJob) error
	ClaimNext(ctx context.Context, instanceID string, leaseUntil time.Time) (*model.OrchestrationJob, error)
	Complete(ctx context.Context, jobID string) error
	Fail(ctx context.Context, jobID string, errMsg string) error
	ReleaseExpiredLeases(ctx context.Context) (int, error)
	HasActiveJobForTask(ctx context.Context, taskID string) (bool, error)
}

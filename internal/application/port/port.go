package port

import (
	"context"

	"danqing-teams/internal/domain/model"
)

type TeamService interface {
	List(ctx context.Context) ([]model.Team, error)
	Get(ctx context.Context, teamID string, controllerView bool) (*model.TeamDetail, error)
	Create(ctx context.Context, req model.CreateTeamRequest) (*model.TeamDetail, error)
	Update(ctx context.Context, teamID string, req model.UpdateTeamRequest) (*model.Team, error)
	Delete(ctx context.Context, teamID string) error
	ListWorkers(ctx context.Context, teamID string, controllerView bool) (any, error)
	UpsertWorker(ctx context.Context, teamID string, req model.UpsertWorkerRequest, workerID string) (*model.WorkerAgent, error)
	DeleteWorker(ctx context.Context, teamID, workerID string) error
	GetController(ctx context.Context, teamID string) (*model.TeamController, error)
	UpdateController(ctx context.Context, teamID string, c model.TeamController) error
	ListHumans(ctx context.Context, teamID string) ([]model.HumanMember, error)
	AddHuman(ctx context.Context, teamID string, h model.HumanMember) error
}

type TaskService interface {
	List(ctx context.Context, teamID string, status model.TaskStatus) ([]model.TeamTask, error)
	Get(ctx context.Context, teamID, taskID string) (*model.TeamTask, error)
	Submit(ctx context.Context, teamID string, req model.SubmitTaskRequest) (*model.TeamTask, error)
	SendMessage(ctx context.Context, teamID string, req model.SendTeamMessageRequest) (*model.SendTeamMessageResponse, error)
	ListMessages(ctx context.Context, teamID, taskID string) ([]model.TeamMessage, error)
	Timeline(ctx context.Context, teamID, taskID string) ([]model.TimelineEvent, error)
	Reports(ctx context.Context, teamID, taskID string) ([]model.Report, error)
	GetPlan(ctx context.Context, teamID, taskID, runID string) (*model.ExecutionPlan, error)
	Cancel(ctx context.Context, teamID, taskID string) error
}

type AgentService interface {
	List(ctx context.Context, role model.AgentRole) ([]model.Agent, error)
	Get(ctx context.Context, agentID string) (*model.Agent, error)
	Create(ctx context.Context, req model.CreateAgentRequest) (*model.Agent, error)
	Update(ctx context.Context, agentID string, req model.UpdateAgentRequest) (*model.Agent, error)
	Delete(ctx context.Context, agentID string) error
	ListTeamMembers(ctx context.Context, teamID string) ([]model.Agent, error)
	AddToTeam(ctx context.Context, teamID, agentID string) error
	RemoveFromTeam(ctx context.Context, teamID, agentID string) error
}

type ApprovalService interface {
	ListPending(ctx context.Context, teamID string) ([]model.ApprovalRequest, error)
	Get(ctx context.Context, teamID, approvalID string) (*model.ApprovalRequest, error)
	Approve(ctx context.Context, teamID, approvalID string, req model.DecideApprovalRequest) (*model.ApprovalRequest, error)
	Reject(ctx context.Context, teamID, approvalID string, req model.DecideApprovalRequest) (*model.ApprovalRequest, error)
}

type TodoService interface {
	List(ctx context.Context, teamID, taskID string) ([]model.TodoItem, error)
	Create(ctx context.Context, teamID, title, taskID string) (*model.TodoItem, error)
	Update(ctx context.Context, teamID, todoID string, done bool) (*model.TodoItem, error)
}

type WorkspaceService interface {
	ListArtifacts(ctx context.Context, teamID string) ([]model.WorkspaceArtifact, error)
	CreateArtifact(ctx context.Context, teamID string, req model.CreateArtifactRequest) (*model.WorkspaceArtifact, error)
	ListKnowledgeDocs(ctx context.Context, teamID, workerID string) ([]model.KnowledgeDoc, error)
	SaveKnowledgeDocs(ctx context.Context, teamID, workerID string, req model.UpsertKnowledgeDocsRequest) error
}

// OrchestrationService is used by the job worker and task submission path.
type OrchestrationService interface {
	SubmitTask(ctx context.Context, teamID string, req model.SubmitTaskRequest) (*model.TeamTask, error)
	SendTeamMessage(ctx context.Context, teamID string, req model.SendTeamMessageRequest) (*model.SendTeamMessageResponse, error)
}

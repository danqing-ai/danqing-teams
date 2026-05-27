package assembler

import (
	"encoding/json"

	"danqing-teams/internal/api/rest/dto"
	"danqing-teams/internal/domain/model"
)

func convert[S, D any](src S) D {
	var dst D
	b, _ := json.Marshal(src)
	_ = json.Unmarshal(b, &dst)
	return dst
}

func ToTeam(m model.Team) dto.Team { return convert[model.Team, dto.Team](m) }
func ToTeams(ms []model.Team) []dto.Team {
	out := make([]dto.Team, len(ms))
	for i, m := range ms {
		out[i] = ToTeam(m)
	}
	return out
}

func ToTeamDetail(m *model.TeamDetail) *dto.TeamDetail {
	if m == nil {
		return nil
	}
	d := convert[model.TeamDetail, dto.TeamDetail](*m)
	return &d
}

func FromCreateTeamRequest(d dto.CreateTeamRequest) model.CreateTeamRequest {
	return convert[dto.CreateTeamRequest, model.CreateTeamRequest](d)
}

func FromUpdateTeamRequest(d dto.UpdateTeamRequest) model.UpdateTeamRequest {
	return convert[dto.UpdateTeamRequest, model.UpdateTeamRequest](d)
}

func FromUpsertWorkerRequest(d dto.UpsertWorkerRequest) model.UpsertWorkerRequest {
	return convert[dto.UpsertWorkerRequest, model.UpsertWorkerRequest](d)
}

func ToWorkerAgent(m *model.WorkerAgent) dto.WorkerAgent {
	if m == nil {
		return dto.WorkerAgent{}
	}
	return convert[model.WorkerAgent, dto.WorkerAgent](*m)
}

func ToWorkerAgents(ms []model.WorkerAgent) []dto.WorkerAgent {
	out := make([]dto.WorkerAgent, len(ms))
	for i, m := range ms {
		out[i] = ToWorkerAgent(&m)
	}
	return out
}

func ToPersonaCatalog(ms []model.WorkerPersonaCatalog) []dto.WorkerPersonaCatalog {
	return convert[[]model.WorkerPersonaCatalog, []dto.WorkerPersonaCatalog](ms)
}

func FromTeamController(d dto.TeamController) model.TeamController {
	return convert[dto.TeamController, model.TeamController](d)
}

func ToTeamController(m *model.TeamController) dto.TeamController {
	if m == nil {
		return dto.TeamController{}
	}
	return convert[model.TeamController, dto.TeamController](*m)
}

func FromHumanMember(d dto.HumanMember) model.HumanMember {
	return convert[dto.HumanMember, model.HumanMember](d)
}

func ToHumanMembers(ms []model.HumanMember) []dto.HumanMember {
	return convert[[]model.HumanMember, []dto.HumanMember](ms)
}

func FromSubmitTaskRequest(d dto.SubmitTaskRequest) model.SubmitTaskRequest {
	return convert[dto.SubmitTaskRequest, model.SubmitTaskRequest](d)
}

func FromSendTeamMessageRequest(d dto.SendTeamMessageRequest) model.SendTeamMessageRequest {
	return convert[dto.SendTeamMessageRequest, model.SendTeamMessageRequest](d)
}

func ToSendTeamMessageResponse(m *model.SendTeamMessageResponse) dto.SendTeamMessageResponse {
	if m == nil {
		return dto.SendTeamMessageResponse{}
	}
	return convert[model.SendTeamMessageResponse, dto.SendTeamMessageResponse](*m)
}

func ToTeamTask(m *model.TeamTask) dto.TeamTask {
	if m == nil {
		return dto.TeamTask{}
	}
	return convert[model.TeamTask, dto.TeamTask](*m)
}

func ToTeamTasks(ms []model.TeamTask) []dto.TeamTask {
	out := make([]dto.TeamTask, len(ms))
	for i, m := range ms {
		out[i] = ToTeamTask(&m)
	}
	return out
}

func ToTeamMessages(ms []model.TeamMessage) []dto.TeamMessage {
	return convert[[]model.TeamMessage, []dto.TeamMessage](ms)
}

func ToTimelineEvents(ms []model.TimelineEvent) []dto.TimelineEvent {
	return convert[[]model.TimelineEvent, []dto.TimelineEvent](ms)
}

func ToExecutionPlan(m *model.ExecutionPlan) dto.ExecutionPlan {
	if m == nil {
		return dto.ExecutionPlan{}
	}
	return convert[model.ExecutionPlan, dto.ExecutionPlan](*m)
}

func ToReports(ms []model.Report) []dto.Report {
	return convert[[]model.Report, []dto.Report](ms)
}

func FromDecideApprovalRequest(d dto.DecideApprovalRequest) model.DecideApprovalRequest {
	return convert[dto.DecideApprovalRequest, model.DecideApprovalRequest](d)
}

func ToApprovalRequest(m *model.ApprovalRequest) dto.ApprovalRequest {
	if m == nil {
		return dto.ApprovalRequest{}
	}
	return convert[model.ApprovalRequest, dto.ApprovalRequest](*m)
}

func ToApprovalRequests(ms []model.ApprovalRequest) []dto.ApprovalRequest {
	out := make([]dto.ApprovalRequest, len(ms))
	for i, m := range ms {
		out[i] = ToApprovalRequest(&m)
	}
	return out
}

func ToTodoItem(m *model.TodoItem) dto.TodoItem {
	if m == nil {
		return dto.TodoItem{}
	}
	return convert[model.TodoItem, dto.TodoItem](*m)
}

func ToTodoItems(ms []model.TodoItem) []dto.TodoItem {
	out := make([]dto.TodoItem, len(ms))
	for i, m := range ms {
		out[i] = ToTodoItem(&m)
	}
	return out
}

func FromCreateArtifactRequest(d dto.CreateArtifactRequest) model.CreateArtifactRequest {
	return convert[dto.CreateArtifactRequest, model.CreateArtifactRequest](d)
}

func ToWorkspaceArtifact(m *model.WorkspaceArtifact) dto.WorkspaceArtifact {
	if m == nil {
		return dto.WorkspaceArtifact{}
	}
	return convert[model.WorkspaceArtifact, dto.WorkspaceArtifact](*m)
}

func ToWorkspaceArtifacts(ms []model.WorkspaceArtifact) []dto.WorkspaceArtifact {
	out := make([]dto.WorkspaceArtifact, len(ms))
	for i, m := range ms {
		out[i] = ToWorkspaceArtifact(&m)
	}
	return out
}

func ToKnowledgeDocs(ms []model.KnowledgeDoc) []dto.KnowledgeDoc {
	return convert[[]model.KnowledgeDoc, []dto.KnowledgeDoc](ms)
}

func FromUpsertKnowledgeDocsRequest(d dto.UpsertKnowledgeDocsRequest) model.UpsertKnowledgeDocsRequest {
	return convert[dto.UpsertKnowledgeDocsRequest, model.UpsertKnowledgeDocsRequest](d)
}

func FromCreateAgentRequest(d dto.CreateAgentRequest) model.CreateAgentRequest {
	return convert[dto.CreateAgentRequest, model.CreateAgentRequest](d)
}

func FromUpdateAgentRequest(d dto.UpdateAgentRequest) model.UpdateAgentRequest {
	return convert[dto.UpdateAgentRequest, model.UpdateAgentRequest](d)
}

func ToAgent(m *model.Agent) dto.Agent {
	if m == nil {
		return dto.Agent{}
	}
	return convert[model.Agent, dto.Agent](*m)
}

func ToAgents(ms []model.Agent) []dto.Agent {
	out := make([]dto.Agent, len(ms))
	for i, m := range ms {
		out[i] = ToAgent(&m)
	}
	return out
}

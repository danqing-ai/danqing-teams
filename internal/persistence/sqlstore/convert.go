package sqlstore

import (
	"time"

	"danqing-teams/internal/contract"
	"gorm.io/gorm/clause"
)

func nowUTC() time.Time { return time.Now().UTC() }

func teamFromRow(r teamRow) contract.Team {
	return contract.Team{
		ID: r.ID, Name: r.Name, Description: r.Description,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func workerFromRow(r workerRow) contract.WorkerAgent {
	return contract.WorkerAgent{
		ID: r.ID, Name: r.Name, Persona: r.Persona,
		Skills: r.Skills, Tools: r.Tools, KnowledgeBase: r.KB,
	}
}

func agentFromRow(r agentRow) contract.Agent {
	return contract.Agent{
		ID: r.ID, Name: r.Name, Description: r.Description, Role: r.Role,
		LLM: contract.AgentLLMConfig{
			URL: r.LLMURL, APIKey: r.LLMAPIKey,
			DefaultModel: r.DefaultModel, AllModels: r.AllModels,
		},
		SystemPrompt: r.SystemPrompt, MinFunctionCallingRounds: r.MinFunctionCallingRounds,
		Skills: r.Skills, Tools: r.Tools, KnowledgeBase: r.KB,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func agentToRow(a *contract.Agent) agentRow {
	return agentRow{
		ID: a.ID, Name: a.Name, Description: a.Description, Role: a.Role,
		LLMURL: a.LLM.URL, LLMAPIKey: a.LLM.APIKey,
		DefaultModel: a.LLM.DefaultModel, AllModels: a.LLM.AllModels,
		SystemPrompt: a.SystemPrompt, MinFunctionCallingRounds: a.MinFunctionCallingRounds,
		Skills: a.Skills, Tools: a.Tools, KB: a.KnowledgeBase,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func taskFromRow(r taskRow) contract.TeamTask {
	return contract.TeamTask{
		ID: r.ID, TeamID: r.TeamID, Content: r.Content,
		Status: r.Status, CloseReason: r.CloseReason,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func dispatchFromRow(r dispatchRow) contract.Dispatch {
	return contract.Dispatch{
		ID: r.ID, TaskID: r.TaskID, WorkerID: r.WorkerID, WorkerName: r.WorkerName,
		Intent: r.Intent, ContextSummary: r.ContextSummary, Round: r.Round, CreatedAt: r.CreatedAt,
	}
}

func runFromRow(r workerRunRow) contract.WorkerRun {
	return contract.WorkerRun{
		ID: r.ID, TaskID: r.TaskID, DispatchID: r.DispatchID,
		WorkerID: r.WorkerID, Status: r.Status,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func planFromRow(r executionPlanRow) contract.ExecutionPlan {
	return contract.ExecutionPlan{
		RunID: r.RunID, SkillIDs: r.SkillIDs, ToolIDs: r.ToolIDs,
		Rationale: r.Rationale, EvaluatedRisk: r.EvaluatedRisk, HighRiskItems: r.HighRiskItems,
	}
}

func reportFromRow(r reportRow) contract.Report {
	return contract.Report{
		ID: r.ID, RunID: r.RunID, TaskID: r.TaskID, WorkerID: r.WorkerID, WorkerName: r.WorkerName,
		ContentMarkdown: r.ContentMarkdown, Intent: r.Intent, SuggestedActions: r.SuggestedActions,
		CreatedAt: r.CreatedAt,
	}
}

func approvalFromRow(r approvalRow) contract.ApprovalRequest {
	return contract.ApprovalRequest{
		ID: r.ID, TeamID: r.TeamID, TaskID: r.TaskID, RunID: r.RunID,
		Summary: r.Summary, HighRiskItems: r.HighRiskItems,
		Status: r.Status, Comment: r.Comment, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func jobFromRow(r orchestrationJobRow) contract.OrchestrationJob {
	j := contract.OrchestrationJob{
		ID: r.ID, TeamID: r.TeamID, TaskID: r.TaskID, Kind: r.Kind,
		Payload: r.Payload, DedupKey: r.DedupKey, Status: r.Status,
		LeaseOwner: r.LeaseOwner, LastError: r.LastError,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.LeaseUntil != nil {
		j.LeaseUntil = *r.LeaseUntil
	}
	return j
}

func upsertColumns(cols ...string) clause.OnConflict {
	return clause.OnConflict{DoUpdates: clause.AssignmentColumns(cols)}
}

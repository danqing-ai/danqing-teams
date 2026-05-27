package mcp

import (
	"context"
	"encoding/json"

	"danqing-teams/internal/api/rest/dto"
	"danqing-teams/internal/application/assembler"
	"danqing-teams/internal/application/port"
	"danqing-teams/internal/domain/model"
)

// Tools bridges MCP tool calls to application services.
type Tools struct {
	Teams     port.TeamService
	Tasks     port.TaskService
	Approvals port.ApprovalService
}

type CallRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type CallResponse struct {
	Content any `json:"content"`
}

func (t *Tools) Call(ctx context.Context, req CallRequest) (CallResponse, error) {
	switch req.Name {
	case "teams_list":
		list, err := t.Teams.List(ctx)
		if err != nil {
			return CallResponse{}, err
		}
		return CallResponse{Content: assembler.ToTeams(list)}, nil
	case "teams_get":
		var args struct {
			TeamID string `json:"teamId"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		team, err := t.Teams.Get(ctx, args.TeamID, false)
		if err != nil {
			return CallResponse{}, err
		}
		return CallResponse{Content: assembler.ToTeamDetail(team)}, nil
	case "workers_upsert":
		var args struct {
			TeamID   string                `json:"teamId"`
			Worker   dto.UpsertWorkerRequest `json:"worker"`
			WorkerID string                `json:"workerId,omitempty"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		w, err := t.Teams.UpsertWorker(ctx, args.TeamID, assembler.FromUpsertWorkerRequest(args.Worker), args.WorkerID)
		if err != nil {
			return CallResponse{}, err
		}
		return CallResponse{Content: assembler.ToWorkerAgent(w)}, nil
	case "task_submit":
		var args struct {
			TeamID  string `json:"teamId"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		task, err := t.Tasks.Submit(ctx, args.TeamID, model.SubmitTaskRequest{Content: args.Content})
		if err != nil {
			return CallResponse{}, err
		}
		return CallResponse{Content: assembler.ToTeamTask(task)}, nil
	case "task_timeline":
		var args struct {
			TeamID string `json:"teamId"`
			TaskID string `json:"taskId"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		events, err := t.Tasks.Timeline(ctx, args.TeamID, args.TaskID)
		if err != nil {
			return CallResponse{}, err
		}
		return CallResponse{Content: assembler.ToTimelineEvents(events)}, nil
	case "task_cancel":
		var args struct {
			TeamID string `json:"teamId"`
			TaskID string `json:"taskId"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		err := t.Tasks.Cancel(ctx, args.TeamID, args.TaskID)
		return CallResponse{Content: ginOK{"status": "cancelled"}}, err
	case "approval_list":
		var args struct {
			TeamID string `json:"teamId"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		list, err := t.Approvals.ListPending(ctx, args.TeamID)
		if err != nil {
			return CallResponse{}, err
		}
		return CallResponse{Content: assembler.ToApprovalRequests(list)}, nil
	case "approval_decide":
		var args struct {
			TeamID     string `json:"teamId"`
			ApprovalID string `json:"approvalId"`
			Approve    bool   `json:"approve"`
			Comment    string `json:"comment,omitempty"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		dec := model.DecideApprovalRequest{Comment: args.Comment}
		if args.Approve {
			a, err := t.Approvals.Approve(ctx, args.TeamID, args.ApprovalID, dec)
			if err != nil {
				return CallResponse{}, err
			}
			return CallResponse{Content: assembler.ToApprovalRequest(a)}, nil
		}
		a, err := t.Approvals.Reject(ctx, args.TeamID, args.ApprovalID, dec)
		if err != nil {
			return CallResponse{}, err
		}
		return CallResponse{Content: assembler.ToApprovalRequest(a)}, nil
	default:
		return CallResponse{}, errsUnknownTool(req.Name)
	}
}

type ginOK map[string]string

type unknownToolError string

func (e unknownToolError) Error() string { return "unknown tool: " + string(e) }

func errsUnknownTool(name string) error { return unknownToolError(name) }

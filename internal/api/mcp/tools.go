package mcp

import (
	"context"
	"encoding/json"

	"danqing-teams/internal/contract"
	"danqing-teams/internal/service"
)

// Tools bridges MCP tool calls to application services.
type Tools struct {
	Teams     *service.TeamService
	Tasks     *service.TaskService
	Approvals *service.ApprovalService
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
		return CallResponse{Content: list}, err
	case "teams_get":
		var args struct {
			TeamID string `json:"teamId"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		team, err := t.Teams.Get(ctx, args.TeamID, false)
		return CallResponse{Content: team}, err
	case "workers_upsert":
		var args struct {
			TeamID string                    `json:"teamId"`
			Worker contract.UpsertWorkerRequest `json:"worker"`
			WorkerID string `json:"workerId,omitempty"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		w, err := t.Teams.UpsertWorker(ctx, args.TeamID, args.Worker, args.WorkerID)
		return CallResponse{Content: w}, err
	case "task_submit":
		var args struct {
			TeamID  string                    `json:"teamId"`
			Content string                    `json:"content"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		task, err := t.Tasks.Submit(ctx, args.TeamID, contract.SubmitTaskRequest{Content: args.Content})
		return CallResponse{Content: task}, err
	case "task_timeline":
		var args struct {
			TeamID string `json:"teamId"`
			TaskID string `json:"taskId"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return CallResponse{}, err
		}
		events, err := t.Tasks.Timeline(ctx, args.TeamID, args.TaskID)
		return CallResponse{Content: events}, err
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
		return CallResponse{Content: list}, err
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
		dec := contract.DecideApprovalRequest{Comment: args.Comment}
		if args.Approve {
			a, err := t.Approvals.Approve(ctx, args.TeamID, args.ApprovalID, dec)
			return CallResponse{Content: a}, err
		}
		a, err := t.Approvals.Reject(ctx, args.TeamID, args.ApprovalID, dec)
		return CallResponse{Content: a}, err
	default:
		return CallResponse{}, errsUnknownTool(req.Name)
	}
}

type ginOK map[string]string

type unknownToolError string

func (e unknownToolError) Error() string { return "unknown tool: " + string(e) }

func errsUnknownTool(name string) error { return unknownToolError(name) }

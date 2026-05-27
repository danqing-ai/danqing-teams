package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"danqing-teams/internal/persistence/memory"
	"danqing-teams/internal/provider/llm/mock"
	"danqing-teams/internal/application/service"
	"danqing-teams/internal/application/service/events"
)

func setupTools(t *testing.T) (*Tools, string) {
	t.Helper()
	store := memory.NewStore()
	_ = memory.SeedDemoTeam(context.Background(), store)
	reg := store.Registry()
	hub := events.NewNoop()
	orch := service.NewOrchestrationService(reg.Teams, reg.Tasks, reg.Approvals, reg.Jobs, mock.New(), hub, true)
	worker := service.NewOrchestrationWorker(orch, reg.Jobs, reg.Recover, "test")
	worker.Start(context.Background())
	teams, _ := reg.Teams.ListTeams(context.Background())
	return &Tools{
		Teams:     service.NewTeamService(reg.Teams),
		Tasks:     service.NewTaskService(reg.Tasks, orch),
		Approvals: service.NewApprovalService(reg.Teams, reg.Tasks, reg.Approvals, hub, orch),
	}, teams[0].ID
}

func TestTools_TeamsList(t *testing.T) {
	tools, _ := setupTools(t)
	resp, err := tools.Call(context.Background(), CallRequest{Name: "teams_list"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content == nil {
		t.Fatal("expected teams list")
	}
}

func TestTools_TaskSubmit(t *testing.T) {
	tools, teamID := setupTools(t)
	args, _ := json.Marshal(map[string]string{"teamId": teamID, "content": "P1 告警"})
	resp, err := tools.Call(context.Background(), CallRequest{Name: "task_submit", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content == nil {
		t.Fatal("expected task")
	}
}

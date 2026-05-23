package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"danqing-teams/internal/persistence/memory"
	"danqing-teams/internal/provider/llm/mock"
	"danqing-teams/internal/service"
	"danqing-teams/internal/service/events"
)

func setupTools(t *testing.T) (*Tools, string) {
	t.Helper()
	store := memory.NewStore()
	_ = memory.SeedDemoTeam(context.Background(), store)
	hub := events.NewNoop()
	orch := service.NewOrchestrationService(store, store, store, store, mock.New(), hub, true)
	worker := service.NewOrchestrationWorker(orch, store, store, "test")
	worker.Start(context.Background())
	teams, _ := store.ListTeams(context.Background())
	return &Tools{
		Teams:     service.NewTeamService(store),
		Tasks:     service.NewTaskService(store, orch),
		Approvals: service.NewApprovalService(store, store, store, hub, orch),
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

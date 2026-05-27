package service

import (
	"context"
	"testing"
	"time"

	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/persistence/memory"
	"danqing-teams/internal/provider/llm/mock"
	"danqing-teams/internal/application/service/events"
)

func TestSendTeamMessage_DispatchesWorker(t *testing.T) {
	store := memory.NewStore()
	_ = memory.SeedDemoTeam(context.Background(), store)
	hub := events.NewNoop()
	orch := NewOrchestrationService(store, store, store, store, mock.New(), hub, true)
	worker := NewOrchestrationWorker(orch, store, store, "test")
	worker.Start(context.Background())
	svc := orch
	teams, _ := store.ListTeams(context.Background())

	resp, err := svc.SendTeamMessage(context.Background(), teams[0].ID, model.SendTeamMessageRequest{
		Content: "线上 CPU 飙高且有多条 P1 告警",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message.Role != model.MessageRoleUser {
		t.Fatalf("message role=%s", resp.Message.Role)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		msgs, _ := store.ListMessages(context.Background(), resp.Task.ID)
		hasController := false
		for _, m := range msgs {
			if m.Role == model.MessageRoleController {
				hasController = true
			}
		}
		got, _ := store.GetTask(context.Background(), "", resp.Task.ID)
		if hasController && (got.Status == model.TaskCompleted || got.Status == model.TaskAwaitingApproval) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("controller message or task completion not observed")
}

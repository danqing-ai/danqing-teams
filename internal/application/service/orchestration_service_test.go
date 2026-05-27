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

func TestOrchestration_SubmitTask_Completes(t *testing.T) {
	store := memory.NewStore()
	_ = memory.SeedDemoTeam(context.Background(), store)
	hub := events.NewNoop()
	orch := NewOrchestrationService(store, store, store, store, mock.New(), hub, true)
	worker := NewOrchestrationWorker(orch, store, store, "test")
	worker.Start(context.Background())
	svc := orch
	teams, _ := store.ListTeams(context.Background())

	task, err := svc.SubmitTask(context.Background(), teams[0].ID, model.SubmitTaskRequest{
		Content: "线上 CPU 飙高且有多条 P1 告警",
	})
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := store.GetTask(context.Background(), "", task.ID)
		if got.Status == model.TaskCompleted || got.Status == model.TaskAwaitingApproval {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("task status=%s", task.Status)
}

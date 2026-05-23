package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"danqing-teams/internal/contract"
	"danqing-teams/internal/persistence/seed"
	"danqing-teams/internal/persistence/sqlstore"
	"danqing-teams/internal/provider/llm/mock"
	"danqing-teams/internal/service/events"
)

func TestOrchestrationWorker_RecoverAfterRestart(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "teams.db")
	store, err := sqlstore.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := seed.DemoTeam(context.Background(), store); err != nil {
		t.Fatal(err)
	}
	teams, err := store.ListTeams(context.Background())
	if err != nil || len(teams) == 0 {
		t.Fatal("expected demo team")
	}
	teamID := teams[0].ID

	task := &contract.TeamTask{
		ID: "task-recover-1", TeamID: teamID, Content: "线上 CPU 飙高且有多条 P1 告警",
		Status: contract.TaskDispatching,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := store.CreateTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	_ = store.AppendMessage(context.Background(), &contract.TeamMessage{
		ID: "msg-1", TeamID: teamID, TaskID: task.ID,
		Role: contract.MessageRoleUser, Content: task.Content,
		CreatedAt: time.Now().UTC(),
	})

	hub := events.NewNoop()
	orch1 := NewOrchestrationService(store, store, store, store, mock.New(), hub, true)
	worker1 := NewOrchestrationWorker(orch1, store, store, "instance-a")
	if err := worker1.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}

	active, err := store.HasActiveJobForTask(context.Background(), task.ID)
	if err != nil || !active {
		t.Fatalf("expected pending job after recover, active=%v err=%v", active, err)
	}
	_ = store.Close()

	store2, err := sqlstore.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store2.Close() })

	orch2 := NewOrchestrationService(store2, store2, store2, store2, mock.New(), hub, true)
	worker2 := NewOrchestrationWorker(orch2, store2, store2, "instance-b")
	worker2.Start(context.Background())

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		got, err := store2.GetTask(context.Background(), teamID, task.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == contract.TaskCompleted {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	got, _ := store2.GetTask(context.Background(), teamID, task.ID)
	t.Fatalf("task not completed after restart, status=%s", got.Status)
}

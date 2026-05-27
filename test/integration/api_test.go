package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"danqing-teams/internal/api/rest"
	"danqing-teams/internal/api/rest/controller"
	"danqing-teams/internal/api/rest/dto"
	"danqing-teams/internal/application/service"
	"danqing-teams/internal/application/service/events"
	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/persistence/memory"
	"danqing-teams/internal/provider/llm/mock"
)

func setupRouter(t *testing.T, autoApprove bool) (*controller.Controller, string) {
	t.Helper()
	store := memory.NewStore()
	if err := memory.SeedDemoTeam(context.Background(), store); err != nil {
		t.Fatal(err)
	}
	reg := store.Registry()
	hub := events.NewNoop()
	orch := service.NewOrchestrationService(reg.Teams, reg.Tasks, reg.Approvals, reg.Jobs, mock.New(), hub, autoApprove)
	worker := service.NewOrchestrationWorker(orch, reg.Jobs, reg.Recover, "test")
	worker.Start(context.Background())
	h := &controller.Controller{
		Teams:     service.NewTeamService(reg.Teams),
		Tasks:     service.NewTaskService(reg.Tasks, orch),
		Approvals: service.NewApprovalService(reg.Teams, reg.Tasks, reg.Approvals, hub, orch),
	}
	teams, _ := reg.Teams.ListTeams(context.Background())
	return h, teams[0].ID
}

func TestSubmitTask_AlertFlow(t *testing.T) {
	h, teamID := setupRouter(t, true)
	r := rest.NewRouter(h, nil)

	body, _ := json.Marshal(dto.SubmitTaskRequest{
		Content: "线上 CPU 飙高且有多条 P1 告警",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+teamID+"/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("submit: %d %s", w.Code, w.Body.String())
	}

	var task dto.TeamTask
	if err := json.Unmarshal(w.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID+"/tasks/"+task.ID, nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		var got dto.TeamTask
		_ = json.Unmarshal(w2.Body.Bytes(), &got)
		if got.Status == dto.TaskCompleted || got.Status == dto.TaskAwaitingApproval {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("task did not complete in time")
}

func TestLLMRemote_NotImplemented(t *testing.T) {
	remote := struct {
		Complete func(context.Context, model.CompletionRequest) (model.CompletionResponse, error)
	}{}
	_ = remote
}

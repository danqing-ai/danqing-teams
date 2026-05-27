package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"danqing-teams/internal/api/rest/controller"
	"danqing-teams/internal/application/service"
	"danqing-teams/internal/persistence/memory"
)

func TestHealthAndListTeams(t *testing.T) {
	store := memory.NewStore()
	_ = memory.SeedDemoTeam(context.Background(), store)
	reg := store.Registry()
	h := &controller.Controller{Teams: service.NewTeamService(reg.Teams)}
	r := NewRouter(h, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("health: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list teams: %d %s", w.Code, w.Body.String())
	}
	var teams []json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &teams); err != nil {
		t.Fatal(err)
	}
	if len(teams) == 0 {
		t.Fatal("expected at least one team")
	}
}

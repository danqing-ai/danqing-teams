package orchestration

import (
	"context"
	"testing"

	"danqing-teams/internal/domain/model"
	"danqing-teams/internal/provider/llm/mock"
)

func TestDispatchWorker_MockController_Alert(t *testing.T) {
	llm := mock.New()
	ctrl := &model.TeamController{SystemPrompt: "match by persona"}
	personas := []model.WorkerPersonaCatalog{
		{Name: "AlertAnalyst", Persona: "告警分析"},
		{Name: "ClusterOperator", Persona: "集群运维扩容"},
	}
	p, ok := DispatchWorker(context.Background(), llm, ctrl, "线上 CPU 飙高 P1 告警", personas)
	if !ok || p.Name != "AlertAnalyst" {
		t.Fatalf("got %+v ok=%v", p, ok)
	}
}

func TestParseControllerDispatch(t *testing.T) {
	personas := []model.WorkerPersonaCatalog{{Name: "ClusterOperator", Persona: "x"}}
	p, ok := parseControllerDispatch("DISPATCH:ClusterOperator\nreason", personas)
	if !ok || p.Name != "ClusterOperator" {
		t.Fatalf("got %+v", p)
	}
}

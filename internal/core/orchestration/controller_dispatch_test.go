package orchestration

import (
	"context"
	"testing"

	"danqing-teams/internal/contract"
	"danqing-teams/internal/provider/llm/mock"
)

func TestDispatchWorker_MockController_Alert(t *testing.T) {
	llm := mock.New()
	ctrl := &contract.TeamController{SystemPrompt: "match by persona"}
	personas := []contract.WorkerPersonaCatalog{
		{Name: "AlertAnalyst", Persona: "告警分析"},
		{Name: "ClusterOperator", Persona: "集群运维扩容"},
	}
	p, ok := DispatchWorker(context.Background(), llm, ctrl, "线上 CPU 飙高 P1 告警", personas)
	if !ok || p.Name != "AlertAnalyst" {
		t.Fatalf("got %+v ok=%v", p, ok)
	}
}

func TestParseControllerDispatch(t *testing.T) {
	personas := []contract.WorkerPersonaCatalog{{Name: "ClusterOperator", Persona: "x"}}
	p, ok := parseControllerDispatch("DISPATCH:ClusterOperator\nreason", personas)
	if !ok || p.Name != "ClusterOperator" {
		t.Fatalf("got %+v", p)
	}
}

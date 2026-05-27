package mock

import (
	"context"
	"strings"
	"testing"

	"danqing-teams/internal/domain/model"
)

func TestProvider_WorkerReply_Alert(t *testing.T) {
	p := New()
	resp, err := p.Complete(context.Background(), model.CompletionRequest{
		Role: model.LLMRoleWorker,
		Context: map[string]string{
			"worker_name": "AlertAnalyst",
			"intent":      "分析 P1 告警",
			"plan_skills": "alert.triage",
			"plan_tools":  "prometheus.query",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Content, "告警") {
		t.Fatalf("expected alert content, got %q", resp.Content)
	}
}

func TestProvider_WorkerReply_Scale(t *testing.T) {
	p := New()
	resp, err := p.Complete(context.Background(), model.CompletionRequest{
		Role: model.LLMRoleWorker,
		Context: map[string]string{
			"worker_name": "ClusterOperator",
			"intent":      "对 prod 执行扩容",
			"plan_skills": "k8s.scale",
			"plan_tools":  "k8s.deployment.scale",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Content, "扩容") && !strings.Contains(resp.Content, "replicas") {
		t.Fatalf("expected scale content, got %q", resp.Content)
	}
}

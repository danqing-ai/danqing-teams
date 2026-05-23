package orchestration

import (
	"testing"

	"danqing-teams/internal/contract"
)

func personas() []contract.WorkerPersonaCatalog {
	return []contract.WorkerPersonaCatalog{
		{ID: "w1", Name: "AlertAnalyst", Persona: "负责告警归因、指标与日志分析；不执行集群变更"},
		{ID: "w2", Name: "ClusterOperator", Persona: "负责 Kubernetes 扩容、节点迁移、集群配置变更"},
		{ID: "w3", Name: "ConfigAuditor", Persona: "负责配置审查与合规检查"},
	}
}

func TestMatchWorker_Alert(t *testing.T) {
	p, ok := MatchWorker("线上 CPU 飙高且有多条 P1 告警", personas())
	if !ok || p.ID != "w1" {
		t.Fatalf("want AlertAnalyst, got %+v ok=%v", p, ok)
	}
}

func TestMatchWorker_ScaleFollowUp(t *testing.T) {
	p, ok := MatchFollowUpWorker("需要集群运维对 prod 执行扩容", personas())
	if !ok || p.ID != "w2" {
		t.Fatalf("want ClusterOperator, got %+v ok=%v", p, ok)
	}
}

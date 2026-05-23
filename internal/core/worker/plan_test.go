package worker

import (
	"testing"

	"danqing-teams/internal/contract"
)

func clusterProfile() contract.WorkerPrivateProfile {
	return contract.WorkerPrivateProfile{
		WorkerID: "w2",
		Skills: []contract.Skill{
			{ID: "cluster.inspect", Name: "Inspect", Keywords: []string{"inspect", "节点"}, RiskLevel: contract.RiskLow},
			{ID: "k8s.scale", Name: "Scale", Keywords: []string{"扩容", "scale"}, RiskLevel: contract.RiskHigh},
		},
		Tools: []contract.ToolBinding{
			{ToolID: "k8s.nodes.list", Name: "List", RiskLevel: contract.RiskLow},
			{ToolID: "k8s.deployment.scale", Name: "Scale Deploy", RiskLevel: contract.RiskHigh},
		},
	}
}

func TestPlanExecution_ListOnly(t *testing.T) {
	plan := PlanExecution("查看节点状态", clusterProfile())
	if len(plan.SkillIDs) == 0 {
		t.Fatal("expected at least one skill")
	}
}

func TestPlanExecution_Scale(t *testing.T) {
	plan := PlanExecution("对 prod 执行扩容", clusterProfile())
	hasHigh := false
	for _, id := range plan.SkillIDs {
		if id == "k8s.scale" {
			hasHigh = true
		}
	}
	for _, id := range plan.ToolIDs {
		if id == "k8s.deployment.scale" {
			hasHigh = true
		}
	}
	if !hasHigh {
		t.Fatalf("expected high-risk scale items in plan: %+v", plan)
	}
}

package worker

import (
	"testing"

	"danqing-teams/internal/domain/model"
)

func clusterProfile() model.WorkerPrivateProfile {
	return model.WorkerPrivateProfile{
		WorkerID: "w2",
		Skills: []model.Skill{
			{ID: "cluster.inspect", Name: "Inspect", Keywords: []string{"inspect", "节点"}, RiskLevel: model.RiskLow},
			{ID: "k8s.scale", Name: "Scale", Keywords: []string{"扩容", "scale"}, RiskLevel: model.RiskHigh},
		},
		Tools: []model.ToolBinding{
			{ToolID: "k8s.nodes.list", Name: "List", RiskLevel: model.RiskLow},
			{ToolID: "k8s.deployment.scale", Name: "Scale Deploy", RiskLevel: model.RiskHigh},
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

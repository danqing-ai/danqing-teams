package policy

import (
	"testing"

	"danqing-teams/internal/domain/model"
)

func TestEvaluatePlan_HighToolOnly(t *testing.T) {
	profile := model.WorkerPrivateProfile{
		Skills: []model.Skill{
			{ID: "cluster.inspect", Name: "Inspect", RiskLevel: model.RiskLow},
			{ID: "k8s.scale", Name: "Scale", RiskLevel: model.RiskHigh},
		},
		Tools: []model.ToolBinding{
			{ToolID: "k8s.nodes.list", Name: "List Nodes", RiskLevel: model.RiskLow},
			{ToolID: "k8s.deployment.scale", Name: "Scale Deploy", RiskLevel: model.RiskHigh},
		},
	}
	plan := model.ExecutionPlan{
		SkillIDs: []string{"cluster.inspect"},
		ToolIDs:  []string{"k8s.nodes.list"},
	}
	max, items := EvaluatePlan(profile, plan)
	if max != model.RiskLow || len(items) != 0 {
		t.Fatalf("inspect-only should be low, got %v %v", max, items)
	}

	plan2 := model.ExecutionPlan{
		SkillIDs: []string{"k8s.scale"},
		ToolIDs:  []string{"k8s.deployment.scale"},
	}
	max2, items2 := EvaluatePlan(profile, plan2)
	if max2 != model.RiskHigh || len(items2) != 2 {
		t.Fatalf("scale plan should be high with 2 items, got %v %v", max2, items2)
	}
}

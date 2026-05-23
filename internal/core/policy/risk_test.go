package policy

import (
	"testing"

	"danqing-teams/internal/contract"
)

func TestEvaluatePlan_HighToolOnly(t *testing.T) {
	profile := contract.WorkerPrivateProfile{
		Skills: []contract.Skill{
			{ID: "cluster.inspect", Name: "Inspect", RiskLevel: contract.RiskLow},
			{ID: "k8s.scale", Name: "Scale", RiskLevel: contract.RiskHigh},
		},
		Tools: []contract.ToolBinding{
			{ToolID: "k8s.nodes.list", Name: "List Nodes", RiskLevel: contract.RiskLow},
			{ToolID: "k8s.deployment.scale", Name: "Scale Deploy", RiskLevel: contract.RiskHigh},
		},
	}
	plan := contract.ExecutionPlan{
		SkillIDs: []string{"cluster.inspect"},
		ToolIDs:  []string{"k8s.nodes.list"},
	}
	max, items := EvaluatePlan(profile, plan)
	if max != contract.RiskLow || len(items) != 0 {
		t.Fatalf("inspect-only should be low, got %v %v", max, items)
	}

	plan2 := contract.ExecutionPlan{
		SkillIDs: []string{"k8s.scale"},
		ToolIDs:  []string{"k8s.deployment.scale"},
	}
	max2, items2 := EvaluatePlan(profile, plan2)
	if max2 != contract.RiskHigh || len(items2) != 2 {
		t.Fatalf("scale plan should be high with 2 items, got %v %v", max2, items2)
	}
}

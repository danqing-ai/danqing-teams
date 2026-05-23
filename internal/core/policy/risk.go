package policy

import "danqing-teams/internal/contract"

// EvaluatePlan aggregates risk from Worker-reported execution plan items only.
func EvaluatePlan(profile contract.WorkerPrivateProfile, plan contract.ExecutionPlan) (contract.RiskLevel, []contract.RiskItem) {
	skillRisk := map[string]contract.Skill{}
	for _, s := range profile.Skills {
		skillRisk[s.ID] = s
	}
	toolRisk := map[string]contract.ToolBinding{}
	for _, t := range profile.Tools {
		toolRisk[t.ToolID] = t
	}

	max := contract.RiskLow
	var highItems []contract.RiskItem

	for _, sid := range plan.SkillIDs {
		if s, ok := skillRisk[sid]; ok {
			max = contract.MaxRisk(max, s.RiskLevel)
			if s.RiskLevel == contract.RiskHigh {
				highItems = append(highItems, contract.RiskItem{
					Type: "skill", ID: s.ID, DisplayName: s.Name,
				})
			}
		}
	}
	for _, tid := range plan.ToolIDs {
		if t, ok := toolRisk[tid]; ok {
			max = contract.MaxRisk(max, t.RiskLevel)
			if t.RiskLevel == contract.RiskHigh {
				highItems = append(highItems, contract.RiskItem{
					Type: "mcp_tool", ID: t.ToolID, DisplayName: t.Name,
				})
			}
		}
	}
	return max, highItems
}

func RequiresApproval(max contract.RiskLevel, highItems []contract.RiskItem) bool {
	return max == contract.RiskHigh && len(highItems) > 0
}

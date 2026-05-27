package policy

import "danqing-teams/internal/domain/model"

// EvaluatePlan aggregates risk from Worker-reported execution plan items only.
func EvaluatePlan(profile model.WorkerPrivateProfile, plan model.ExecutionPlan) (model.RiskLevel, []model.RiskItem) {
	skillRisk := map[string]model.Skill{}
	for _, s := range profile.Skills {
		skillRisk[s.ID] = s
	}
	toolRisk := map[string]model.ToolBinding{}
	for _, t := range profile.Tools {
		toolRisk[t.ToolID] = t
	}

	max := model.RiskLow
	var highItems []model.RiskItem

	for _, sid := range plan.SkillIDs {
		if s, ok := skillRisk[sid]; ok {
			max = model.MaxRisk(max, s.RiskLevel)
			if s.RiskLevel == model.RiskHigh {
				highItems = append(highItems, model.RiskItem{
					Type: "skill", ID: s.ID, DisplayName: s.Name,
				})
			}
		}
	}
	for _, tid := range plan.ToolIDs {
		if t, ok := toolRisk[tid]; ok {
			max = model.MaxRisk(max, t.RiskLevel)
			if t.RiskLevel == model.RiskHigh {
				highItems = append(highItems, model.RiskItem{
					Type: "mcp_tool", ID: t.ToolID, DisplayName: t.Name,
				})
			}
		}
	}
	return max, highItems
}

func RequiresApproval(max model.RiskLevel, highItems []model.RiskItem) bool {
	return max == model.RiskHigh && len(highItems) > 0
}

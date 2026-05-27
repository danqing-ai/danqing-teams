package worker

import (
	"strings"

	"danqing-teams/internal/domain/model"
)

// PlanExecution selects skills and MCP tools from private profile based on intent.
// Controller never sees the selected IDs.
func PlanExecution(intent string, profile model.WorkerPrivateProfile) model.ExecutionPlan {
	intentLower := strings.ToLower(intent)
	var skillIDs, toolIDs []string
	var rationale strings.Builder

	for _, s := range profile.Skills {
		if skillMatches(intentLower, s) {
			skillIDs = append(skillIDs, s.ID)
		}
	}
	for _, t := range profile.Tools {
		if toolMatches(intentLower, t) {
			toolIDs = append(toolIDs, t.ToolID)
		}
	}

	// Default read-only inspect if nothing matched
	if len(skillIDs) == 0 && len(toolIDs) == 0 {
		for _, s := range profile.Skills {
			if s.RiskLevel == model.RiskLow {
				skillIDs = append(skillIDs, s.ID)
				break
			}
		}
		for _, t := range profile.Tools {
			if t.RiskLevel == model.RiskLow {
				toolIDs = append(toolIDs, t.ToolID)
				break
			}
		}
		rationale.WriteString("未命中明确关键词，使用默认只读技能/工具。")
	} else {
		rationale.WriteString("根据意图在私有档案中匹配技能与 MCP Tool。")
	}

	return model.ExecutionPlan{
		SkillIDs:  skillIDs,
		ToolIDs:   toolIDs,
		Rationale: rationale.String(),
	}
}

func skillMatches(intent string, s model.Skill) bool {
	for _, kw := range s.Keywords {
		if strings.Contains(intent, strings.ToLower(kw)) {
			return true
		}
	}
	if strings.Contains(intent, strings.ToLower(s.ID)) {
		return true
	}
	return false
}

func toolMatches(intent string, t model.ToolBinding) bool {
	if strings.Contains(intent, strings.ToLower(t.ToolID)) {
		return true
	}
	if strings.Contains(intent, "扩容") && strings.Contains(t.ToolID, "scale") {
		return true
	}
	if strings.Contains(intent, "节点") && strings.Contains(t.ToolID, "node") {
		return true
	}
	if (strings.Contains(intent, "告警") || strings.Contains(intent, "alert")) &&
		(strings.Contains(t.ToolID, "prometheus") || strings.Contains(t.ToolID, "alert")) {
		return true
	}
	return false
}

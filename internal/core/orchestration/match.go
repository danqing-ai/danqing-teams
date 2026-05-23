package orchestration

import (
	"strings"

	"danqing-teams/internal/contract"
)

// MatchWorker picks a worker using persona text only (no skills/tools).
func MatchWorker(taskContent string, personas []contract.WorkerPersonaCatalog) (contract.WorkerPersonaCatalog, bool) {
	if len(personas) == 0 {
		return contract.WorkerPersonaCatalog{}, false
	}
	task := strings.ToLower(taskContent)
	best := personas[0]
	bestScore := -1

	for _, p := range personas {
		score := scorePersona(task, strings.ToLower(p.Persona), strings.ToLower(p.Name))
		if score > bestScore {
			bestScore = score
			best = p
		}
	}
	return best, bestScore > 0
}

func scorePersona(task, persona, name string) int {
	score := 0
	keywords := []struct {
		terms []string
		pts   int
	}{
		{[]string{"告警", "alert", "p1", "cpu", "指标"}, 10},
		{[]string{"扩容", "scale", "迁移", "节点", "集群", "kubernetes", "k8s"}, 10},
		{[]string{"配置", "config", "diff", "合规", "审计"}, 8},
	}
	for _, k := range keywords {
		for _, term := range k.terms {
			if strings.Contains(task, term) && (strings.Contains(persona, term) || strings.Contains(name, term)) {
				score += k.pts
			}
		}
	}
	// persona-specific boosts
	if strings.Contains(name, "alert") && (strings.Contains(task, "告警") || strings.Contains(task, "alert")) {
		score += 15
	}
	if strings.Contains(name, "cluster") && (strings.Contains(task, "扩容") || strings.Contains(task, "scale")) {
		score += 15
	}
	if strings.Contains(name, "config") && strings.Contains(task, "配置") {
		score += 12
	}
	return score
}

// MatchFollowUpWorker uses targetPersonaHint from report.
func MatchFollowUpWorker(hint string, personas []contract.WorkerPersonaCatalog) (contract.WorkerPersonaCatalog, bool) {
	return MatchWorker(hint, personas)
}

// AnalyzeReportIntent derives follow-up from mock report patterns (conservative to avoid loops).
func AnalyzeReportIntent(reportMarkdown string) (contract.ReportIntent, []contract.SuggestedAction) {
	lower := strings.ToLower(reportMarkdown)
	// Only explicit handoff phrases trigger follow-up, not mere mentions in narrative.
	if strings.Contains(lower, "handoff: cluster_operator") ||
		strings.Contains(lower, "建议由集群运维") {
		return contract.ReportNeedsFollowUp, []contract.SuggestedAction{{
			Description:       "评估 prod 扩容",
			TargetPersonaHint: "Kubernetes 集群运维 扩容",
		}}
	}
	return contract.ReportFinal, nil
}

// BuildContextSummary produces controller-safe summary without other workers' private data.
func BuildContextSummary(taskContent string, round int) string {
	if round == 0 {
		return "用户提交任务：" + truncate(taskContent, 200)
	}
	return "跟进轮次 " + itoa(round) + "；前序结论已摘要，不含其他 Worker 私有工具/KB 细节。"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

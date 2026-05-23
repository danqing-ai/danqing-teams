package mock

import (
	"context"
	"fmt"
	"strings"

	"danqing-teams/internal/contract"
)

// Provider implements contract.LLMProvider with deterministic rule-based responses.
type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) Complete(_ context.Context, req contract.CompletionRequest) (contract.CompletionResponse, error) {
	switch req.Role {
	case contract.LLMRoleController:
		return contract.CompletionResponse{Content: p.controllerReply(req)}, nil
	case contract.LLMRoleWorker:
		return contract.CompletionResponse{Content: p.workerReply(req)}, nil
	default:
		return contract.CompletionResponse{Content: "ok"}, nil
	}
}

func (p *Provider) controllerReply(req contract.CompletionRequest) string {
	intent := strings.ToLower(strings.TrimSpace(req.Prompt))
	personas := req.Context["personas"]

	// Mock Team Controller: pick worker by persona keywords (same semantics as core.MatchWorker).
	type candidate struct {
		name  string
		score int
	}
	var picks []candidate
	for _, line := range strings.Split(personas, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		parts := strings.SplitN(line[2:], ":", 2)
		if len(parts) < 1 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		persona := ""
		if len(parts) > 1 {
			persona = strings.ToLower(parts[1])
		}
		score := scoreControllerIntent(intent, name, persona)
		if score > 0 {
			picks = append(picks, candidate{name: name, score: score})
		}
	}
	best := ""
	bestScore := -1
	for _, c := range picks {
		if c.score > bestScore {
			bestScore = c.score
			best = c.name
		}
	}
	if best == "" {
		return "DISPATCH:AlertAnalyst"
	}
	return "DISPATCH:" + best
}

func scoreControllerIntent(task, name, persona string) int {
	score := 0
	if strings.Contains(task, "告警") || strings.Contains(task, "alert") || strings.Contains(task, "cpu") {
		if strings.Contains(name, "Alert") || strings.Contains(persona, "告警") {
			score += 20
		}
	}
	if strings.Contains(task, "扩容") || strings.Contains(task, "scale") || strings.Contains(task, "集群") {
		if strings.Contains(name, "Cluster") || strings.Contains(persona, "集群") {
			score += 20
		}
	}
	if strings.Contains(task, "配置") || strings.Contains(task, "config") || strings.Contains(task, "diff") {
		if strings.Contains(name, "Config") || strings.Contains(persona, "配置") {
			score += 18
		}
	}
	return score
}

func (p *Provider) workerReply(req contract.CompletionRequest) string {
	workerName := req.Context["worker_name"]
	intent := req.Context["intent"]
	planSkills := req.Context["plan_skills"]
	planTools := req.Context["plan_tools"]
	var b strings.Builder
	fmt.Fprintf(&b, "## %s 执行报告\n\n", workerName)
	fmt.Fprintf(&b, "**任务意图**：%s\n\n", intent)
	if planSkills != "" || planTools != "" {
		b.WriteString("**本次使用**\n")
		if planSkills != "" {
			fmt.Fprintf(&b, "- 技能：%s\n", planSkills)
		}
		if planTools != "" {
			fmt.Fprintf(&b, "- MCP Tools：%s\n", planTools)
		}
		b.WriteString("\n")
	}

	switch {
	case strings.Contains(strings.ToLower(intent), "告警") || strings.Contains(strings.ToLower(workerName), "alert"):
		b.WriteString("### 结论\n")
		b.WriteString("- 已关联 P1 告警与 CPU 指标异常\n")
		b.WriteString("- 初步根因：某 Deployment 副本 CPU 饱和\n")
		b.WriteString("\n### 建议后续\n")
		b.WriteString("建议由集群运维同学评估是否对 prod 执行扩容。\n")
		b.WriteString("\n<!-- handoff: cluster_operator -->\n")
	case strings.Contains(strings.ToLower(intent), "扩容") || strings.Contains(strings.ToLower(intent), "scale"):
		b.WriteString("### 变更计划\n")
		b.WriteString("- 目标：prod `api-server` Deployment\n")
		b.WriteString("- 操作：replicas 3 → 5\n")
		b.WriteString("\n### 结论\n")
		b.WriteString("变更计划已生成；高危 MCP Tool 已在审批后执行（Mock）。\n")
	default:
		b.WriteString("### 结论\n")
		b.WriteString("任务已按 Worker 私有技能与工具完成（Mock）。\n")
	}

	return b.String()
}

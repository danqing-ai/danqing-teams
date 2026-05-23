package orchestration

import (
	"context"
	"strings"

	"danqing-teams/internal/contract"
)

// DispatchWorker selects a worker via Team Controller (mock LLM) with rule-based fallback.
func DispatchWorker(
	ctx context.Context,
	llm contract.LLMProvider,
	controller *contract.TeamController,
	intent string,
	personas []contract.WorkerPersonaCatalog,
) (contract.WorkerPersonaCatalog, bool) {
	if len(personas) == 0 {
		return contract.WorkerPersonaCatalog{}, false
	}

	personaLine := formatPersonaCatalog(personas)
	system := controller.SystemPrompt
	if system == "" {
		system = "你是 Team Controller，仅依据 Worker 人设匹配分派，不知道 Worker 的技能与 MCP Tool。"
	}

	resp, err := llm.Complete(ctx, contract.CompletionRequest{
		Role:   contract.LLMRoleController,
		Prompt: intent,
		Context: map[string]string{
			"system_prompt": system,
			"personas":      personaLine,
		},
	})
	if err == nil {
		if picked, ok := parseControllerDispatch(resp.Content, personas); ok {
			return picked, true
		}
	}

	return MatchWorker(intent, personas)
}

func formatPersonaCatalog(personas []contract.WorkerPersonaCatalog) string {
	var b strings.Builder
	for _, p := range personas {
		b.WriteString("- ")
		b.WriteString(p.Name)
		b.WriteString(": ")
		b.WriteString(p.Persona)
		b.WriteByte('\n')
	}
	return b.String()
}

func parseControllerDispatch(content string, personas []contract.WorkerPersonaCatalog) (contract.WorkerPersonaCatalog, bool) {
	const marker = "DISPATCH:"
	idx := strings.Index(strings.ToUpper(content), marker)
	if idx < 0 {
		return contract.WorkerPersonaCatalog{}, false
	}
	name := strings.TrimSpace(content[idx+len(marker):])
	if i := strings.IndexAny(name, "\n\r"); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimSpace(name)
	for _, p := range personas {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return contract.WorkerPersonaCatalog{}, false
}

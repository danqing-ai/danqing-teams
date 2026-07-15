package llm

import "danqing-teams/core/domain"

var effortToAnthropicBudget = map[string]int{
	"low":    4_000,
	"medium": 8_000,
	"high":   16_000,
	"xhigh":  24_000,
	"max":    32_000,
}

func ApplyReasoningEffort(provider domain.LLMProviderType, effort string, body map[string]any) {
	if effort == "" || effort == "off" {
		return
	}
	switch provider {
	case domain.LLMProviderAnthropic:
		budget, ok := effortToAnthropicBudget[effort]
		if !ok {
			return
		}
		body["thinking"] = map[string]any{
			"type":          "enabled",
			"budget_tokens": budget,
		}
	default:
		body["reasoning_effort"] = effort
	}
}

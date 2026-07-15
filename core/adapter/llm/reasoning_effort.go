package llm

import (
	"strings"

	"danqing-teams/core/domain"
)

var effortToAnthropicBudget = map[string]int{
	"low":    4_000,
	"medium": 8_000,
	"high":   16_000,
	"xhigh":  24_000,
	"max":    32_000,
}

func isAnthropicAdaptiveModel(model string) bool {
	for _, m := range anthropicAdaptiveModels {
		if strings.Contains(model, m) {
			return true
		}
	}
	return false
}

var anthropicAdaptiveModels = []string{
	"claude-sonnet-5",
	"claude-fable-5",
	"claude-opus-4-7",
	"claude-opus-4-8",
	"claude-sonnet-4-6",
	"claude-opus-4-6",
}

func ApplyReasoningEffort(provider domain.LLMProviderType, model, effort string, body map[string]any) {
	if effort == "" || effort == "off" {
		return
	}
	switch provider {
	case domain.LLMProviderAnthropic:
		if isAnthropicAdaptiveModel(model) {
			body["thinking"] = map[string]any{
				"type": "adaptive",
			}
			body["effort"] = effort
			body["display"] = "summarized"
		} else {
			budget, ok := effortToAnthropicBudget[effort]
			if !ok {
				return
			}
			body["thinking"] = map[string]any{
				"type":          "enabled",
				"budget_tokens": budget,
			}
		}
	default:
		body["reasoning_effort"] = effort
	}
}

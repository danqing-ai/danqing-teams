package runtime

import (
	"context"
	"strings"
	"sync"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

// ModelLimitsRegistry provides model context window and max output token
// lookups. Config-based entries (from the YAML file) take priority over
// the built-in hardcoded defaults.
type ModelLimitsRegistry struct {
	mu      sync.RWMutex
	config  []domain.ModelLimit
	byModel map[string]domain.ModelLimit // lazy index
}

// NewModelLimitsRegistry creates a registry seeded with built-in defaults.
func NewModelLimitsRegistry() *ModelLimitsRegistry {
	return &ModelLimitsRegistry{}
}

// SetLimits replaces the config-based model limits.
func (r *ModelLimitsRegistry) SetLimits(limits []domain.ModelLimit) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = limits
	r.byModel = nil // invalidate index
}

func (r *ModelLimitsRegistry) ensureIndex() {
	if r.byModel != nil {
		return
	}
	r.byModel = make(map[string]domain.ModelLimit, len(r.config))
	for _, l := range r.config {
		r.byModel[strings.ToLower(l.Model)] = l
	}
}

// ContextWindow returns the context window size for a given model ID.
// Lookup order: config overrides → pattern rules → hardcoded defaults.
func (r *ModelLimitsRegistry) ContextWindow(modelID string) int {
	_, model := splitModelID(modelID)
	if model == "" {
		return defaultModelContextWindow
	}
	lower := strings.ToLower(model)

	// 1. Config-based exact match
	r.mu.RLock()
	r.ensureIndex()
	if l, ok := r.byModel[lower]; ok {
		r.mu.RUnlock()
		return l.ContextWindow
	}
	r.mu.RUnlock()

	// 2. Pattern matching (hardcoded rules)
	for _, rule := range contextWindowRules {
		if strings.Contains(lower, rule.pattern) {
			return rule.window
		}
	}

	return defaultModelContextWindow
}

// MaxOutputTokens returns the maximum output tokens for a given model.
func (r *ModelLimitsRegistry) MaxOutputTokens(modelID string) int {
	_, model := splitModelID(modelID)
	if model == "" {
		return defaultMaxOutputTokens
	}
	lower := strings.ToLower(model)

	// 1. Config-based exact match
	r.mu.RLock()
	r.ensureIndex()
	if l, ok := r.byModel[lower]; ok {
		r.mu.RUnlock()
		if l.MaxOutput > 0 {
			return l.MaxOutput
		}
		// If config entry exists but maxOutput is 0, still fall through to rules
	}
	r.mu.RUnlock()

	// 2. Pattern matching (hardcoded rules)
	for _, rule := range maxOutputRules {
		if strings.Contains(lower, rule.pattern) {
			return rule.window
		}
	}

	return defaultMaxOutputTokens
}

// AllLimits returns the current config-based model limits.
func (r *ModelLimitsRegistry) AllLimits() []domain.ModelLimit {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.ModelLimit, len(r.config))
	copy(out, r.config)
	return out
}

// LoadFromConfig reads model limits from the config store.
func (r *ModelLimitsRegistry) LoadFromConfig(ctx context.Context, store port.ConfigStore) {
	if store == nil {
		return
	}
	cfg, err := store.Load(ctx)
	if err != nil {
		return
	}
	r.SetLimits(cfg.LLM.ModelLimits)
}

func splitModelID(modelID string) (provider, model string) {
	idx := strings.Index(modelID, "/")
	if idx < 0 {
		return "", modelID
	}
	return modelID[:idx], modelID[idx+1:]
}

const (
	defaultModelContextWindow = 128_000
	defaultMaxOutputTokens    = 8_192
)

// --- Pattern-match rules (hardcoded fallback) ---

type patternRule struct {
	pattern string
	window  int
}

var contextWindowRules = []patternRule{
	// OpenAI
	{"gpt-4.1", 1_047_576},
	{"gpt-4o", 128_000},
	{"gpt-4-turbo", 128_000},
	{"o3-mini", 200_000},
	{"o3", 200_000},
	{"o4-mini", 200_000},
	{"o1-pro", 200_000},
	{"o1-mini", 128_000},
	{"o1", 200_000},

	// Anthropic
	{"claude-opus", 200_000},
	{"claude-sonnet", 200_000},
	{"claude-haiku", 200_000},
	{"claude-3", 200_000},

	// DeepSeek
	{"deepseek-v4", 1_000_000},
	{"deepseek-v3", 64_000},
	{"deepseek-r1", 64_000},
	{"deepseek-chat", 64_000},

	// Google
	{"gemini-2.5", 1_048_576},
	{"gemini-2.0", 1_048_576},
	{"gemini-1.5-pro", 2_097_152},
	{"gemini-1.5-flash", 1_048_576},

	// Zhipu
	{"glm-4-long", 1_000_000},
	{"glm-4", 128_000},

	// Qwen
	{"qwen-long", 10_000_000},
	{"qwen-max", 32_000},
	{"qwen3", 131_072},
	{"qwen-plus", 131_072},
	{"qwen-turbo", 131_072},

	// Moonshot
	{"moonshot-v1-128k", 128_000},
	{"moonshot-v1-32k", 32_000},
	{"moonshot-v1-8k", 8_000},
	{"kimi", 128_000},
}

var maxOutputRules = []patternRule{
	{"gpt-4.1", 32_768},
	{"gpt-4o", 16_384},
	{"o3", 100_000},
	{"o4-mini", 100_000},
	{"o1", 100_000},
	{"claude-opus", 32_000},
	{"claude-sonnet", 64_000},
	{"claude-haiku", 8_192},
	{"claude-3", 4_096},
	{"deepseek-v4", 384_000},
	{"deepseek-r1", 8_192},
	{"deepseek-chat", 8_192},
	{"gemini-2.5", 65_536},
	{"gemini-2.0", 8_192},
	{"gemini", 8_192},
	{"glm-4", 4_096},
	{"qwen", 8_192},
	{"moonshot", 4_096},
	{"kimi", 4_096},
}

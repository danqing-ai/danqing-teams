package service

import (
	"context"
	"strings"
	"sync"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

// ModelConfigRegistry provides per-model lookups for context window, max output
// tokens, and generation parameters — all sourced from the YAML config file.
// When a model is not found in config, hardcoded defaults are used as fallback.
type ModelConfigRegistry struct {
	mu      sync.RWMutex
	config  []domain.ModelConfig
	byModel map[string]domain.ModelConfig // lazy index
}

// NewModelConfigRegistry creates an empty registry.
func NewModelConfigRegistry() *ModelConfigRegistry {
	return &ModelConfigRegistry{}
}

// SetModels replaces the config-based model entries.
func (r *ModelConfigRegistry) SetModels(models []domain.ModelConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = models
	r.byModel = nil // invalidate index
}

func (r *ModelConfigRegistry) ensureIndex() {
	if r.byModel != nil {
		return
	}
	r.byModel = make(map[string]domain.ModelConfig, len(r.config))
	for _, l := range r.config {
		r.byModel[strings.ToLower(l.Model)] = l
	}
}

// lookup returns the config entry for a model ID, or nil if not found.
func (r *ModelConfigRegistry) lookup(modelID string) *domain.ModelConfig {
	_, model := splitModelID(modelID)
	if model == "" {
		return nil
	}
	r.mu.RLock()
	r.ensureIndex()
	c, ok := r.byModel[strings.ToLower(model)]
	r.mu.RUnlock()
	if !ok {
		return nil
	}
	return &c
}

// ContextWindow returns the context window size for a given model ID.
// Lookup order: config exact match → default constant (128K).
func (r *ModelConfigRegistry) ContextWindow(modelID string) int {
	if c := r.lookup(modelID); c != nil && c.ContextWindow > 0 {
		return c.ContextWindow
	}
	return defaultModelContextWindow
}

// MaxOutputTokens returns the maximum output tokens for a given model.
// Lookup order: config exact match → default constant (8K).
func (r *ModelConfigRegistry) MaxOutputTokens(modelID string) int {
	if c := r.lookup(modelID); c != nil && c.MaxOutput > 0 {
		return c.MaxOutput
	}
	return defaultMaxOutputTokens
}

// GenParams returns generation parameter overrides for a given model.
// Lookup order: config exact match → nil (provider default).
func (r *ModelConfigRegistry) GenParams(modelID string) *port.ModelGenParams {
	c := r.lookup(modelID)
	if c == nil {
		return nil
	}
	return modelConfigToGenParams(*c)
}

func modelConfigToGenParams(c domain.ModelConfig) *port.ModelGenParams {
	// Return nil if all values are zero (no overrides).
	if c.Temperature == 0 && c.TopP == 0 && c.FrequencyPenalty == 0 && c.PresencePenalty == 0 && len(c.Stop) == 0 {
		return nil
	}
	return &port.ModelGenParams{
		Temperature:      c.Temperature,
		TopP:             c.TopP,
		FrequencyPenalty: c.FrequencyPenalty,
		PresencePenalty:  c.PresencePenalty,
		Stop:             c.Stop,
	}
}

// AllModels returns the current config-based model configs.
func (r *ModelConfigRegistry) AllModels() []domain.ModelConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.ModelConfig, len(r.config))
	copy(out, r.config)
	return out
}

// LoadFromConfig reads model configs from the config store.
func (r *ModelConfigRegistry) LoadFromConfig(ctx context.Context, store port.ConfigStore) {
	if store == nil {
		return
	}
	cfg, err := store.Load(ctx)
	if err != nil {
		return
	}
	r.SetModels(cfg.LLM.Models)
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

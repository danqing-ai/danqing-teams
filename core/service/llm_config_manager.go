package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

type LLMConfigManager struct {
	store  port.LLMConfigRepo
	mu     sync.RWMutex
	cache  map[string]domain.LLMProviderConfig
	loaded bool
}

func NewLLMConfigManager(store port.LLMConfigRepo) *LLMConfigManager {
	return &LLMConfigManager{store: store}
}

func (m *LLMConfigManager) ensureCache(ctx context.Context) {
	m.mu.RLock()
	if m.loaded {
		m.mu.RUnlock()
		return
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.loaded {
		return
	}
	cfgs, err := m.store.GetAll(ctx)
	if err != nil {
		return
	}
	m.cache = make(map[string]domain.LLMProviderConfig, len(cfgs))
	for _, cfg := range cfgs {
		m.cache[cfg.ID] = cfg
	}
	m.loaded = true
}

func (m *LLMConfigManager) GetAll(ctx context.Context) ([]domain.LLMProviderConfig, error) {
	m.ensureCache(ctx)
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.LLMProviderConfig, 0, len(m.cache))
	for _, cfg := range m.cache {
		out = append(out, cfg)
	}
	return out, nil
}

func (m *LLMConfigManager) GetByID(ctx context.Context, id string) (domain.LLMProviderConfig, error) {
	m.ensureCache(ctx)
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cache[id], nil
}

func (m *LLMConfigManager) Upsert(ctx context.Context, req domain.UpsertLLMProviderConfigRequest) (domain.LLMProviderConfig, error) {
	if req.Provider == "" {
		return domain.LLMProviderConfig{}, fmt.Errorf("provider required")
	}
	if req.Name == "" {
		return domain.LLMProviderConfig{}, fmt.Errorf("name required")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	m.mu.RLock()
	existing := m.cache[fmt.Sprintf("llm-%s", req.Name)]
	m.mu.RUnlock()

	cfg := domain.LLMProviderConfig{
		ID:        existing.ID,
		Provider:  req.Provider,
		Name:      req.Name,
		APIKey:    req.APIKey,
		BaseURL:   req.BaseURL,
		Models:    req.Models,
		UpdatedAt: now,
	}
	if cfg.ID == "" {
		cfg.ID = fmt.Sprintf("llm-%s", req.Name)
		cfg.CreatedAt = now
	}

	if err := m.store.Upsert(ctx, cfg); err != nil {
		return domain.LLMProviderConfig{}, err
	}

	m.mu.Lock()
	if m.cache == nil {
		m.cache = make(map[string]domain.LLMProviderConfig)
	}
	m.cache[cfg.ID] = cfg
	m.loaded = true
	m.mu.Unlock()

	return cfg, nil
}

func (m *LLMConfigManager) Delete(ctx context.Context, id string) error {
	if err := m.store.Delete(ctx, id); err != nil {
		return err
	}
	m.mu.Lock()
	if m.cache != nil {
		delete(m.cache, id)
	}
	m.mu.Unlock()
	return nil
}

func (m *LLMConfigManager) ListModels(ctx context.Context) []domain.LLMModel {
	m.ensureCache(ctx)
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []domain.LLMModel
	for _, cfg := range m.cache {
		for _, ref := range cfg.Models {
			if !ref.Enabled {
				continue
			}
			modelID := cfg.Name + "/" + ref.Name
			var efforts []string
			if cfg.Provider == domain.LLMProviderAnthropic {
				efforts = domain.DefaultEffortsAnthropic
			} else {
				efforts = domain.DefaultEffortsOpenAI
			}
			out = append(out, domain.LLMModel{
				ID:               modelID,
				Name:             ref.Name,
				ProviderID:       cfg.ID,
				Provider:         string(cfg.Provider),
				Enabled:          ref.Enabled,
				AvailableEfforts: efforts,
			})
		}
	}
	return out
}

func (m *LLMConfigManager) ResolveProvider(ctx context.Context, modelID string) (domain.LLMProviderConfig, string, error) {
	parts := strings.SplitN(modelID, "/", 3)
	if len(parts) < 2 {
		return domain.LLMProviderConfig{}, "", fmt.Errorf("invalid model identifier: %s, expected format: provider_name/model_name", modelID)
	}
	providerName := parts[0]
	modelName := parts[1]

	m.ensureCache(ctx)
	m.mu.RLock()
	cfg, ok := m.cache[fmt.Sprintf("llm-%s", providerName)]
	m.mu.RUnlock()
	if !ok || cfg.ID == "" {
		return domain.LLMProviderConfig{}, "", fmt.Errorf("provider not found: %s", providerName)
	}
	return cfg, modelName, nil
}

func (m *LLMConfigManager) FetchModels(ctx context.Context, configID string) ([]domain.LLMModelRef, error) {
	m.ensureCache(ctx)
	m.mu.RLock()
	cfg, ok := m.cache[configID]
	m.mu.RUnlock()
	if !ok || cfg.ID == "" {
		return nil, fmt.Errorf("config not found: %s", configID)
	}

	remoteModels, err := listRemoteModels(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("fetch models from remote: %w", err)
	}

	existingEnabled := make(map[string]bool, len(cfg.Models))
	for _, ref := range cfg.Models {
		existingEnabled[ref.Name] = ref.Enabled
	}

	now := time.Now().UTC().Format(time.RFC3339)
	merged := make([]domain.LLMModelRef, 0, len(remoteModels))
	for _, name := range remoteModels {
		enabled := existingEnabled[name]
		merged = append(merged, domain.LLMModelRef{Name: name, Enabled: enabled})
	}

	cfg.Models = merged
	cfg.UpdatedAt = now

	if err := m.store.Upsert(ctx, cfg); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	m.mu.Lock()
	if m.cache != nil {
		m.cache[cfg.ID] = cfg
	}
	m.mu.Unlock()

	return merged, nil
}

func (m *LLMConfigManager) FetchModelsFromRequest(ctx context.Context, req domain.UpsertLLMProviderConfigRequest) ([]domain.LLMModelRef, error) {
	cfg := domain.LLMProviderConfig{
		Provider: req.Provider,
		APIKey:   req.APIKey,
		BaseURL:  req.BaseURL,
	}
	remoteModels, err := listRemoteModels(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("fetch models from remote: %w", err)
	}
	refs := make([]domain.LLMModelRef, 0, len(remoteModels))
	for _, name := range remoteModels {
		refs = append(refs, domain.LLMModelRef{Name: name, Enabled: true})
	}
	return refs, nil
}

func (m *LLMConfigManager) ToggleModel(ctx context.Context, configID, modelName string, enabled bool) (domain.LLMProviderConfig, error) {
	m.ensureCache(ctx)
	m.mu.RLock()
	cfg, ok := m.cache[configID]
	m.mu.RUnlock()
	if !ok || cfg.ID == "" {
		return domain.LLMProviderConfig{}, fmt.Errorf("config not found: %s", configID)
	}

	found := false
	for i, ref := range cfg.Models {
		if ref.Name == modelName {
			cfg.Models[i].Enabled = enabled
			found = true
			break
		}
	}
	if !found {
		return domain.LLMProviderConfig{}, fmt.Errorf("model not found: %s", modelName)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	cfg.UpdatedAt = now

	if err := m.store.Upsert(ctx, cfg); err != nil {
		return domain.LLMProviderConfig{}, fmt.Errorf("save config: %w", err)
	}

	m.mu.Lock()
	if m.cache != nil {
		m.cache[cfg.ID] = cfg
	}
	m.mu.Unlock()

	return cfg, nil
}

var knownAnthropicModels = []string{
	"claude-sonnet-4-20250514",
	"claude-3-opus-20240229",
	"claude-3-sonnet-20240229",
	"claude-3-haiku-20240307",
	"claude-3-5-sonnet-20240620",
	"claude-3-5-sonnet-20241022",
	"claude-3-5-haiku-20241022",
	"claude-3-opus-latest",
	"claude-3-sonnet-latest",
	"claude-3-haiku-latest",
}

func listRemoteModels(ctx context.Context, cfg domain.LLMProviderConfig) ([]string, error) {
	switch cfg.Provider {
	case domain.LLMProviderMock:
		return []string{"mock-gpt-4", "mock-claude"}, nil
	case domain.LLMProviderAnthropic:
		out := make([]string, len(knownAnthropicModels))
		copy(out, knownAnthropicModels)
		return out, nil
	default:
		return listOpenAICompatibleModels(ctx, cfg.BaseURL, cfg.APIKey)
	}
}

func listOpenAICompatibleModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	url := strings.TrimRight(baseURL, "/") + "/models"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse models response: %w", err)
	}

	var names []string
	for _, m := range result.Data {
		if m.ID != "" {
			names = append(names, m.ID)
		}
	}
	return names, nil
}

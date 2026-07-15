package llm

import (
	"context"
	"fmt"
	"strings"

	"danqing-teams/core/service"
	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

type DefaultLLMProviderClient struct {
	mgr     *service.LLMConfigManager
	modelCfg *service.ModelConfigRegistry
}

func NewDefaultLLMProvider(mgr *service.LLMConfigManager, modelCfg *service.ModelConfigRegistry) port.LLMProvider {
	return &DefaultLLMProviderClient{mgr: mgr, modelCfg: modelCfg}
}

func (c *DefaultLLMProviderClient) Chat(ctx context.Context, req port.LLMChatRequest) (port.LLMChatResponse, error) {
	providerName, modelName, effort := ParseModelID(req.Model)
	if providerName == "" {
		return port.LLMChatResponse{}, fmt.Errorf("model not specified or invalid format (expected provider/model)")
	}

	cfg, mn, err := c.mgr.ResolveProvider(ctx, providerName+"/"+modelName)
	if err != nil {
		return port.LLMChatResponse{}, err
	}
	if mn == "" {
		mn = modelName
	}
	req.Model = mn

	// Auto-fill GenParams from registry if not already set by caller.
	if req.GenParams == nil && c.modelCfg != nil {
		req.GenParams = c.modelCfg.GenParams(providerName + "/" + modelName)
	}

	switch cfg.Provider {
	case domain.LLMProviderAnthropic:
		return NewAnthropicProvider(cfg.BaseURL, cfg.APIKey).Chat(ctx, req, effort)
	case domain.LLMProviderMock:
		return NewMock().Chat(ctx, req)
	default:
		return NewHTTPProvider(cfg.BaseURL, cfg.APIKey).Chat(ctx, req, effort)
	}
}

func ParseModelID(modelID string) (providerName, modelName, effort string) {
	parts := strings.SplitN(modelID, "/", 3)
	if len(parts) >= 1 {
		providerName = parts[0]
	}
	if len(parts) >= 2 {
		modelName = parts[1]
	}
	if len(parts) >= 3 {
		effort = parts[2]
	}
	return
}

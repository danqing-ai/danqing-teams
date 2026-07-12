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
	mgr *service.LLMConfigManager
}

func NewDefaultLLMProvider(mgr *service.LLMConfigManager) port.LLMProvider {
	return &DefaultLLMProviderClient{mgr: mgr}
}

func (c *DefaultLLMProviderClient) Chat(ctx context.Context, req port.LLMChatRequest) (port.LLMChatResponse, error) {
	providerName, modelName := ParseModelID(req.Model)
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

	switch cfg.Provider {
	case domain.LLMProviderAnthropic:
		return NewAnthropicProvider(cfg.BaseURL, cfg.APIKey).Chat(ctx, req)
	case domain.LLMProviderMock:
		return NewMock().Chat(ctx, req)
	default:
		return NewHTTPProvider(cfg.BaseURL, cfg.APIKey).Chat(ctx, req)
	}
}

func ParseModelID(modelID string) (providerName, modelName string) {
	parts := strings.SplitN(modelID, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", modelID
}

package service

import (
	"context"
	"fmt"
	"time"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

type MCPManager struct {
	repo port.MCPServerRepo
}

func NewMCPManager(repo port.MCPServerRepo) *MCPManager {
	return &MCPManager{repo: repo}
}

func (m *MCPManager) List(ctx context.Context) ([]domain.MCPServer, error) {
	return m.repo.List(ctx)
}

func (m *MCPManager) Get(ctx context.Context, id string) (domain.MCPServer, error) {
	return m.repo.Get(ctx, id)
}

func (m *MCPManager) Create(ctx context.Context, req domain.UpsertMCPServerRequest) (domain.MCPServer, error) {
	if req.Name == "" {
		return domain.MCPServer{}, fmt.Errorf("MCP server name is required")
	}
	s := domain.MCPServer{
		ID:           fmt.Sprintf("mcp-%d", time.Now().UnixNano()),
		Name:         req.Name,
		Description:  req.Description,
		Transport:    req.Transport,
		Command:      req.Command,
		Args:         req.Args,
		URL:          req.URL,
		Env:          req.Env,
		Headers:      req.Headers,
		EnabledTools: req.EnabledTools,
		ToolTimeout:  req.ToolTimeout,
		Status:       "disconnected",
		Enabled:      req.Enabled,
	}
	if s.Transport == "" {
		if s.Command != "" {
			s.Transport = "stdio"
		} else if s.URL != "" {
			s.Transport = "streamable-http"
		}
	}
	if s.ToolTimeout <= 0 {
		s.ToolTimeout = 300
	}
	if len(s.EnabledTools) == 0 {
		s.EnabledTools = []string{"*"}
	}
	if err := m.repo.Upsert(ctx, s); err != nil {
		return domain.MCPServer{}, err
	}
	return s, nil
}

func (m *MCPManager) Update(ctx context.Context, id string, req domain.UpsertMCPServerRequest) (domain.MCPServer, error) {
	existing, err := m.repo.Get(ctx, id)
	if err != nil {
		return domain.MCPServer{}, fmt.Errorf("MCP server not found: %w", err)
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	existing.Description = req.Description
	existing.Transport = req.Transport
	existing.Command = req.Command
	existing.Args = req.Args
	existing.URL = req.URL
	existing.Env = req.Env
	existing.Headers = req.Headers
	if len(req.EnabledTools) > 0 {
		existing.EnabledTools = req.EnabledTools
	}
	if req.ToolTimeout > 0 {
		existing.ToolTimeout = req.ToolTimeout
	}
	if req.Status != "" {
		existing.Status = req.Status
	}
	existing.Enabled = req.Enabled
	if err := m.repo.Upsert(ctx, existing); err != nil {
		return domain.MCPServer{}, err
	}
	return existing, nil
}

func (m *MCPManager) Delete(ctx context.Context, id string) error {
	return m.repo.Delete(ctx, id)
}

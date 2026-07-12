package service

import (
	"context"
	"fmt"
	"sync"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

type AgentManager struct {
	store          port.AgentRepo
	mu             sync.RWMutex
	cache          map[string]*domain.Agent
	templateLoader func(id string) (*domain.Agent, error)
}

func NewAgentManager(store port.AgentRepo) *AgentManager {
	return &AgentManager{store: store, cache: make(map[string]*domain.Agent)}
}

func (m *AgentManager) SetTemplateLoader(fn func(id string) (*domain.Agent, error)) {
	m.templateLoader = fn
}

func (m *AgentManager) Get(ctx context.Context, id string) (*domain.Agent, error) {
	m.mu.RLock()
	if a, ok := m.cache[id]; ok {
		m.mu.RUnlock()
		return a, nil
	}
	m.mu.RUnlock()
	a, err := m.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.cache[id] = &a
	m.mu.Unlock()
	return &a, nil
}

func (m *AgentManager) List(ctx context.Context) ([]domain.Agent, error) {
	return m.store.List(ctx)
}

func (m *AgentManager) Upsert(ctx context.Context, a domain.Agent) error {
	if err := m.store.Upsert(ctx, a); err != nil {
		return err
	}
	m.mu.Lock()
	m.cache[a.ID] = &a
	m.mu.Unlock()
	return nil
}

func (m *AgentManager) Delete(ctx context.Context, id string) error {
	if err := m.store.Delete(ctx, id); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.cache, id)
	m.mu.Unlock()
	return nil
}

func (m *AgentManager) ResetFromTemplate(ctx context.Context, id string) (*domain.Agent, error) {
	if m.templateLoader == nil {
		return nil, fmt.Errorf("template loader not configured")
	}
	_, err := m.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agent %q not found, cannot reset: %w", id, err)
	}
	tmpl, err := m.templateLoader(id)
	if err != nil {
		return nil, fmt.Errorf("no template found for agent %q: %w", id, err)
	}
	if err := m.store.Upsert(ctx, *tmpl); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.cache[id] = tmpl
	m.mu.Unlock()
	return tmpl, nil
}

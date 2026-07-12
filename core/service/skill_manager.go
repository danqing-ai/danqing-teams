package service

import (
	"context"
	"sync"
	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

type SkillManager struct {
	store      port.SkillRepo
	mu         sync.RWMutex
	cache      map[string]*domain.Skill
	cachedList bool
	listCache  []domain.Skill
}

func NewSkillManager(store port.SkillRepo) *SkillManager {
	return &SkillManager{store: store, cache: make(map[string]*domain.Skill)}
}

func (m *SkillManager) List(ctx context.Context) ([]domain.Skill, error) {
	m.mu.RLock()
	if m.cachedList {
		result := m.listCache
		m.mu.RUnlock()
		return result, nil
	}
	m.mu.RUnlock()
	list, err := m.store.List(ctx)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.listCache = list
	m.cachedList = true
	m.mu.Unlock()
	return list, nil
}

func (m *SkillManager) Get(ctx context.Context, id string) (*domain.Skill, error) {
	m.mu.RLock()
	if s, ok := m.cache[id]; ok {
		m.mu.RUnlock()
		return s, nil
	}
	m.mu.RUnlock()
	list, err := m.store.List(ctx)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	for i := range list {
		m.cache[list[i].ID] = &list[i]
	}
	m.mu.Unlock()
	if s, ok := m.cache[id]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *SkillManager) Upsert(ctx context.Context, s domain.Skill) error {
	if err := m.store.Upsert(ctx, s); err != nil {
		return err
	}
	m.mu.Lock()
	m.cache[s.ID] = &s
	m.cachedList = false
	m.mu.Unlock()
	return nil
}

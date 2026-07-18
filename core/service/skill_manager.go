package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"danqing-teams/core/domain"
	"danqing-teams/core/port"
)

// ValidSkillResourcePrefixes are the allowed top-level dirs for skill resource files.
var ValidSkillResourcePrefixes = []string{"scripts/", "references/", "assets/"}

// NormalizeSkillResourcePath cleans and validates a skill resource relative path.
func NormalizeSkillResourcePath(path string) (string, error) {
	p := strings.TrimSpace(path)
	p = strings.TrimPrefix(p, "/")
	p = strings.ReplaceAll(p, "\\", "/")
	if p == "" {
		return "", fmt.Errorf("path required")
	}
	if strings.Contains(p, "..") {
		return "", fmt.Errorf("invalid path: must not contain \"..\"")
	}
	ok := false
	for _, prefix := range ValidSkillResourcePrefixes {
		if strings.HasPrefix(p, prefix) {
			ok = true
			break
		}
	}
	if !ok {
		return "", fmt.Errorf("invalid resource path %q: must be under scripts/, references/, or assets/", p)
	}
	if strings.HasSuffix(p, "/") {
		return "", fmt.Errorf("invalid path: must be a file, not a directory")
	}
	return p, nil
}

type SkillManager struct {
	store             port.SkillRepo
	filesRepo         port.SkillFileRepo
	mu                sync.RWMutex
	cache             map[string]*domain.Skill
	cachedList        bool
	listCache         []domain.Skill
	templateLoader    func(id string) (*domain.Skill, error)
	fileTemplateLoader func(id string) ([]domain.SkillFile, error)
}

func NewSkillManager(store port.SkillRepo, filesRepo port.SkillFileRepo) *SkillManager {
	return &SkillManager{
		store:     store,
		filesRepo: filesRepo,
		cache:     make(map[string]*domain.Skill),
	}
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
	sk, err := m.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.cache[id] = &sk
	m.mu.Unlock()
	return &sk, nil
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

func (m *SkillManager) Delete(ctx context.Context, id string) error {
	if err := m.store.Delete(ctx, id); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.cache, id)
	m.cachedList = false
	m.mu.Unlock()
	return nil
}

func (m *SkillManager) Files(ctx context.Context, skillID string) ([]domain.SkillFile, error) {
	return m.filesRepo.ListBySkill(ctx, skillID)
}

func (m *SkillManager) File(ctx context.Context, skillID, path string) (domain.SkillFile, error) {
	return m.filesRepo.Get(ctx, skillID, path)
}

func (m *SkillManager) UpsertFile(ctx context.Context, f domain.SkillFile) error {
	path, err := NormalizeSkillResourcePath(f.Path)
	if err != nil {
		return err
	}
	f.Path = path
	if f.SkillID == "" {
		return fmt.Errorf("skillId required")
	}
	if f.ID == "" {
		f.ID = f.SkillID + ":" + f.Path
	}
	if f.Size == 0 {
		f.Size = int64(len(f.Content))
	}
	return m.filesRepo.Upsert(ctx, f)
}

func (m *SkillManager) DeleteFile(ctx context.Context, skillID, path string) error {
	path, err := NormalizeSkillResourcePath(path)
	if err != nil {
		return err
	}
	return m.filesRepo.Delete(ctx, skillID, path)
}

func (m *SkillManager) DeleteFiles(ctx context.Context, skillID string) error {
	return m.filesRepo.DeleteBySkill(ctx, skillID)
}

func (m *SkillManager) SetTemplateLoader(fn func(id string) (*domain.Skill, error)) {
	m.templateLoader = fn
}

func (m *SkillManager) SetFileTemplateLoader(fn func(id string) ([]domain.SkillFile, error)) {
	m.fileTemplateLoader = fn
}

func (m *SkillManager) ResetFromTemplate(ctx context.Context, id string) (*domain.Skill, error) {
	if m.templateLoader == nil {
		return nil, fmt.Errorf("template loader not configured")
	}
	_, err := m.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("skill %q not found: %w", id, err)
	}
	tmpl, err := m.templateLoader(id)
	if err != nil {
		return nil, fmt.Errorf("no template found for skill %q: %w", id, err)
	}
	tmpl.Builtin = true
	if err := m.store.Upsert(ctx, *tmpl); err != nil {
		return nil, err
	}
	// Reset resource files from template if available
	if m.fileTemplateLoader != nil {
		files, err := m.fileTemplateLoader(id)
		if err == nil && len(files) > 0 {
			_ = m.filesRepo.DeleteBySkill(ctx, id)
			for _, f := range files {
				_ = m.filesRepo.Upsert(ctx, f)
			}
		}
	}
	m.mu.Lock()
	m.cache[id] = tmpl
	m.cachedList = false
	m.mu.Unlock()
	return tmpl, nil
}

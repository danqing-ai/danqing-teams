package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"danqing-teams/internal/domain/repository"
	"danqing-teams/internal/persistence/memory"
	"danqing-teams/internal/persistence/seed"
	"danqing-teams/internal/persistence/sqlstore"
)

// Open creates persistence backends from TEAMS_STORE (memory|sqlite) and TEAMS_DB_PATH.
func Open(ctx context.Context, storeKind, dbPath string) (repository.Registry, string, func() error, error) {
	switch strings.ToLower(storeKind) {
	case "", "sqlite":
		if dbPath == "" {
			dbPath = "./data/teams.db"
		}
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return repository.Registry{}, "", nil, err
		}
		s, err := sqlstore.Open(dbPath)
		if err != nil {
			return repository.Registry{}, "", nil, err
		}
		if err := seed.DemoTeam(ctx, s); err != nil {
			_ = s.Close()
			return repository.Registry{}, "", nil, err
		}
		return s.Registry(), "sqlite", s.Close, nil
	case "memory":
		s := memory.NewStore()
		if err := seed.DemoTeam(ctx, s); err != nil {
			return repository.Registry{}, "", nil, err
		}
		return s.Registry(), "memory", func() error { return nil }, nil
	default:
		return repository.Registry{}, "", nil, fmt.Errorf("unknown TEAMS_STORE: %q", storeKind)
	}
}

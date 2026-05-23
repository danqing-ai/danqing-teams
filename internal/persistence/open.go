package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"danqing-teams/internal/persistence/memory"
	"danqing-teams/internal/persistence/seed"
	"danqing-teams/internal/persistence/sqlstore"
)

// Open creates persistence backends from TEAMS_STORE (memory|sqlite) and TEAMS_DB_PATH.
func Open(ctx context.Context, storeKind, dbPath string) (repos any, kind string, closeFn func() error, err error) {
	switch strings.ToLower(storeKind) {
	case "", "sqlite":
		if dbPath == "" {
			dbPath = "./data/teams.db"
		}
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return nil, "", nil, err
		}
		s, err := sqlstore.Open(dbPath)
		if err != nil {
			return nil, "", nil, err
		}
		if err := seed.DemoTeam(ctx, s); err != nil {
			_ = s.Close()
			return nil, "", nil, err
		}
		return s, "sqlite", s.Close, nil
	case "memory":
		s := memory.NewStore()
		if err := seed.DemoTeam(ctx, s); err != nil {
			return nil, "", nil, err
		}
		return s, "memory", func() error { return nil }, nil
	default:
		return nil, "", nil, fmt.Errorf("unknown TEAMS_STORE: %q", storeKind)
	}
}

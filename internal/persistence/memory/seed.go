package memory

import (
	"context"

	"danqing-teams/internal/persistence/seed"
)

// SeedDemoTeam creates the SRE demo team if store is empty.
func SeedDemoTeam(ctx context.Context, s *Store) error {
	return seed.DemoTeam(ctx, s)
}

package memory

import "danqing-teams/internal/domain/repository"

func (s *Store) Registry() repository.Registry {
	return repository.Registry{
		Teams:     s,
		Tasks:     s,
		Workspace: s,
		Approvals: s,
		Agents:    s,
		Jobs:      s,
		Recover:   s,
	}
}

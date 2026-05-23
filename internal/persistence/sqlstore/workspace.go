package sqlstore

import (
	"context"
	"database/sql"

	"danqing-teams/internal/contract"
	"danqing-teams/pkg/id"
)

func (s *Store) ListArtifacts(ctx context.Context, teamID string) ([]contract.WorkspaceArtifact, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, team_id, task_id, title, kind, content, created_at FROM artifacts WHERE team_id = ? ORDER BY created_at DESC`,
		teamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.WorkspaceArtifact
	for rows.Next() {
		var a contract.WorkspaceArtifact
		var created string
		if err := rows.Scan(&a.ID, &a.TeamID, &a.TaskID, &a.Title, &a.Kind, &a.Content, &created); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = parseTime(created)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) CreateArtifact(ctx context.Context, teamID string, a contract.WorkspaceArtifact) (*contract.WorkspaceArtifact, error) {
	if a.ID == "" {
		a.ID = id.New()
	}
	a.TeamID = teamID
	a.CreatedAt = nowUTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO artifacts (id, team_id, task_id, title, kind, content, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.ID, teamID, a.TaskID, a.Title, a.Kind, a.Content, formatTime(a.CreatedAt),
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) ListKnowledgeDocs(ctx context.Context, teamID, workerID string) ([]contract.KnowledgeDoc, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, size FROM knowledge_docs WHERE team_id = ? AND worker_id = ?`, teamID, workerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []contract.KnowledgeDoc
	for rows.Next() {
		var d contract.KnowledgeDoc
		if err := rows.Scan(&d.ID, &d.Title, &d.Size); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) SaveKnowledgeDocs(ctx context.Context, teamID, workerID string, docs []contract.KnowledgeDoc) error {
	return withTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_docs WHERE team_id = ? AND worker_id = ?`, teamID, workerID); err != nil {
			return err
		}
		for _, d := range docs {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO knowledge_docs (team_id, worker_id, id, title, size) VALUES (?, ?, ?, ?, ?)`,
				teamID, workerID, d.ID, d.Title, d.Size,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

package model

import "time"

type WorkspaceArtifact struct {
	ID        string
	TeamID    string
	TaskID    string
	Title     string
	Kind      string // report | note | pin
	Content   string
	CreatedAt time.Time
}

type CreateArtifactRequest struct {
	Title   string
	Kind    string
	Content string
	TaskID  string
}

type KnowledgeDoc struct {
	ID    string
	Title string
	Size  int
}

type UpsertKnowledgeDocsRequest struct {
	Docs []KnowledgeDoc
}

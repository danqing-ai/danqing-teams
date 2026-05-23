package contract

import "time"

type WorkspaceArtifact struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"teamId"`
	TaskID    string    `json:"taskId,omitempty"`
	Title     string    `json:"title"`
	Kind      string    `json:"kind"` // report | note | pin
	Content   string    `json:"content,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateArtifactRequest struct {
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	Content string `json:"content,omitempty"`
	TaskID  string `json:"taskId,omitempty"`
}

type KnowledgeDoc struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Size  int    `json:"size"`
}

type UpsertKnowledgeDocsRequest struct {
	Docs []KnowledgeDoc `json:"docs"`
}

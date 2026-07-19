package domain

type Skill struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AllowedTools  string            `json:"allowedTools,omitempty"`
	Keywords      []string          `json:"keywords"`
	ToolIDs       []string          `json:"toolIds"`
	SystemHint    string            `json:"systemHint"`
	Body          string            `json:"body"`
	SourcePath    string            `json:"sourcePath,omitempty"`
	Builtin       bool              `json:"builtin,omitempty"`
	// TemplateDiverged is computed (not persisted): true when a builtin skill's
	// stored content or resource files differ from the embedded template.
	TemplateDiverged bool `json:"templateDiverged,omitempty"`
}

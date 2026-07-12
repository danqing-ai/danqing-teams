package domain

type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	ToolIDs     []string `json:"toolIds"`
	SystemHint  string   `json:"systemHint"`
}

package domain

type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	RiskLevel   RiskLevel      `json:"riskLevel"`
}

type ToolResult struct {
	Content string         `json:"content"`
	Meta    map[string]any `json:"meta,omitempty"`
}

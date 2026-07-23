package domain

type AgentMode string

const (
	AgentModePrimary  AgentMode = "primary"
	AgentModeSubagent AgentMode = "subagent"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Agent struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Persona      string    `json:"persona"`
	Mode         AgentMode `json:"mode"`
	SystemPrompt string    `json:"systemPrompt"`
	// Steps is the max tool/LLM steps per turn. 0 means follow
	// runtime.turn.max_steps_default.
	Steps        int           `json:"steps"`
	SkillIDs     []string      `json:"skillIds"`
	Tools        []ToolBinding `json:"tools"`
	KnowledgeIDs []string      `json:"knowledgeIds"`
	CanDelegate  bool          `json:"canDelegate"`
	// Builtin is computed (not a DB column): true when an embedded template exists.
	Builtin bool `json:"builtin,omitempty"`
	// MarketSource is computed from YAML frontmatter stored in SystemPrompt
	// (metadata.market.source); not a DB column.
	MarketSource string `json:"marketSource,omitempty"`
}

type ToolBinding struct {
	ToolID    string    `json:"toolId"`
	MCPServer string    `json:"mcpServer"`
	RiskLevel RiskLevel `json:"riskLevel"`
}

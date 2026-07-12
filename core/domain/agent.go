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
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Persona      string       `json:"persona"`
	Mode         AgentMode    `json:"mode"`
	SystemPrompt string       `json:"systemPrompt"`
	Steps        int          `json:"steps"`
	SkillIDs     []string     `json:"skillIds"`
	Tools        []ToolBinding `json:"tools"`
	KnowledgeIDs []string     `json:"knowledgeIds"`
}

type ToolBinding struct {
	ToolID    string    `json:"toolId"`
	MCPServer string    `json:"mcpServer"`
	RiskLevel RiskLevel `json:"riskLevel"`
}

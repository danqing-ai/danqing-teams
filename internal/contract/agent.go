package contract

// WorkerPersonaCatalog is visible to TeamController for matching only.
type WorkerPersonaCatalog struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Persona string `json:"persona"`
}

type Skill struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Keywords    []string  `json:"keywords"`
	RiskLevel   RiskLevel `json:"riskLevel"`
}

type ToolBinding struct {
	ToolID    string    `json:"toolId"`
	MCPServer string    `json:"mcpServer"`
	Name      string    `json:"name"`
	RiskLevel RiskLevel `json:"riskLevel"`
}

type KnowledgeBaseRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// WorkerPrivateProfile is only loaded by worker execution layer.
type WorkerPrivateProfile struct {
	WorkerID      string           `json:"workerId"`
	Skills        []Skill          `json:"skills"`
	Tools         []ToolBinding    `json:"tools"`
	KnowledgeBase KnowledgeBaseRef `json:"knowledgeBase"`
}

type TeamController struct {
	Persona      string `json:"persona"`
	SystemPrompt string `json:"systemPrompt"`
}

type WorkerAgent struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Persona       string           `json:"persona"`
	Skills        []Skill          `json:"skills,omitempty"`
	Tools         []ToolBinding    `json:"tools,omitempty"`
	KnowledgeBase KnowledgeBaseRef `json:"knowledgeBase"`
}

type HumanMember struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email,omitempty"`
	Role        string `json:"role"` // approver | observer
}

package model

// WorkerPersonaCatalog is visible to TeamController for matching only.
type WorkerPersonaCatalog struct {
	ID      string
	Name    string
	Persona string
}

type Skill struct {
	ID          string
	Name        string
	Description string
	Keywords    []string
	RiskLevel   RiskLevel
}

type ToolBinding struct {
	ToolID    string
	MCPServer string
	Name      string
	RiskLevel RiskLevel
}

type KnowledgeBaseRef struct {
	ID   string
	Name string
}

// WorkerPrivateProfile is only loaded by worker execution layer.
type WorkerPrivateProfile struct {
	WorkerID      string
	Skills        []Skill
	Tools         []ToolBinding
	KnowledgeBase KnowledgeBaseRef
}

type TeamController struct {
	Persona      string
	SystemPrompt string
}

type WorkerAgent struct {
	ID            string
	Name          string
	Persona       string
	Skills        []Skill
	Tools         []ToolBinding
	KnowledgeBase KnowledgeBaseRef
}

type HumanMember struct {
	ID          string
	DisplayName string
	Email       string
	Role        string // approver | observer
}

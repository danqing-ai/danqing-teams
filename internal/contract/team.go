package contract

import "time"

type Team struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TeamDetail struct {
	Team
	Controller TeamController   `json:"controller"`
	Workers    []WorkerAgent    `json:"workers,omitempty"`
	Personas   []WorkerPersonaCatalog `json:"personas,omitempty"`
	Humans     []HumanMember    `json:"humans,omitempty"`
}

type CreateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type UpdateTeamRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type UpsertWorkerRequest struct {
	Name          string           `json:"name"`
	Persona       string           `json:"persona"`
	Skills        []Skill          `json:"skills"`
	Tools         []ToolBinding    `json:"tools"`
	KnowledgeBase KnowledgeBaseRef `json:"knowledgeBase"`
}

type TodoItem struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"teamId"`
	TaskID    string    `json:"taskId,omitempty"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}

package model

import "time"

type Team struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TeamDetail struct {
	Team
	Controller TeamController
	Workers    []WorkerAgent
	Personas   []WorkerPersonaCatalog
	Humans     []HumanMember
}

type CreateTeamRequest struct {
	Name        string
	Description string
}

type UpdateTeamRequest struct {
	Name        *string
	Description *string
}

type UpsertWorkerRequest struct {
	Name          string
	Persona       string
	Skills        []Skill
	Tools         []ToolBinding
	KnowledgeBase KnowledgeBaseRef
}

type TodoItem struct {
	ID        string
	TeamID    string
	TaskID    string
	Title     string
	Done      bool
	CreatedAt time.Time
}

package domain

import "time"

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Directory string    `json:"directory"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateProjectRequest struct {
	Name      string `json:"name"`
	Directory string `json:"directory,omitempty"`
}

type UpdateProjectRequest struct {
	Name      string `json:"name,omitempty"`
	Directory string `json:"directory,omitempty"`
}

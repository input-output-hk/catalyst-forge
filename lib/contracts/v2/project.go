package v2

import (
	"time"

	"github.com/google/uuid"
)

// ProjectResponse represents a project response (read-only from v2 API)
type ProjectResponse struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	ProjectKey  string    `json:"project_key"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectListFilter represents filters for listing projects
type ProjectListFilter struct {
	RepoID      *string `json:"repo_id,omitempty" form:"repo_id" binding:"omitempty,uuid4"`
	ProjectKey  *string `json:"project_key,omitempty" form:"project_key"`
	Name        *string `json:"name,omitempty" form:"name"`
	Active      *bool   `json:"active,omitempty" form:"active"`
	TimeRange
	Pagination
	Sort
}

// ProjectIDParam represents a project ID parameter
type ProjectIDParam struct {
	ProjectID uuid.UUID `uri:"project_id" binding:"required,uuid4"`
}

// ProjectKeyParam represents a project key parameter
type ProjectKeyParam struct {
	RepoID     uuid.UUID `uri:"repo_id" binding:"required,uuid4"`
	ProjectKey string    `uri:"project_key" binding:"required"`
}
package contracts

import (
	"time"

	"github.com/google/uuid"
)

// ProjectResponse represents a project response (read-only from v2 API)
type ProjectResponse struct {
	ID                   string    `json:"id"`
	RepoID               string    `json:"repo_id"`
	Path                 string    `json:"path"`                                  // Repo-relative directory for project root
	Slug                 string    `json:"slug"`
	DisplayName          *string   `json:"display_name,omitempty"`
	Status               string    `json:"status"` // "active" or "removed"
	BlueprintFingerprint *string   `json:"blueprint_fingerprint,omitempty"`
	FirstSeenCommit      *string   `json:"first_seen_commit,omitempty"`
	LastSeenCommit       *string   `json:"last_seen_commit,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// ProjectListFilter represents filters for listing projects
type ProjectListFilter struct {
	RepoID *string `json:"repo_id,omitempty" form:"repo_id" binding:"omitempty,uuid4"`
	Path   *string `json:"path,omitempty" form:"path"`
	Slug   *string `json:"slug,omitempty" form:"slug"`
	Status *string `json:"status,omitempty" form:"status" binding:"omitempty,oneof=active removed"`
	TimeRange
	Pagination
	Sort
}

// ProjectIDParam represents a project ID parameter
type ProjectIDParam struct {
	ProjectID uuid.UUID `uri:"project_id" binding:"required,uuid4"`
}

// ProjectPathParam represents a project path parameter
type ProjectPathParam struct {
	RepoID string `uri:"repo_id" binding:"required,uuid4"`
	Path   string `json:"path" form:"path" binding:"required"`
}
package contracts

import (
	"time"

	"github.com/google/uuid"
)

// RepositoryResponse represents a repository response (read-only from v2 API)
type RepositoryResponse struct {
	ID        string    `json:"id"`
	Host      string    `json:"host"`
	Org       string    `json:"org"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RepositoryListFilter represents filters for listing repositories
type RepositoryListFilter struct {
	Host *string `json:"host,omitempty" form:"host"`
	Org  *string `json:"org,omitempty" form:"org"`
	Name *string `json:"name,omitempty" form:"name"`
	TimeRange
	Pagination
	Sort
}

// RepositoryIDParam represents a repository ID parameter
type RepositoryIDParam struct {
	RepositoryID uuid.UUID `uri:"repo_id" binding:"required,uuid4"`
}

// RepositoryPathParam represents a repository path parameter
type RepositoryPathParam struct {
	Host string `uri:"host" binding:"required"`
	Org  string `uri:"org" binding:"required"`
	Name string `uri:"name" binding:"required"`
}
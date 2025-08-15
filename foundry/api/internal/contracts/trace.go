package contracts

import (
	"time"

	"github.com/google/uuid"
)

// TraceCreate represents a request to create a trace
type TraceCreate struct {
	Purpose        string  `json:"purpose" binding:"required,oneof=release deployment build test"`
	RetentionClass string  `json:"retention_class" binding:"required,oneof=short long permanent"`
	RepoID         *string `json:"repo_id,omitempty" binding:"omitempty,uuid4"`
	Branch         *string `json:"branch,omitempty"`
	CreatedBy      *string `json:"created_by,omitempty"`
}

// TraceResponse represents a trace response
type TraceResponse struct {
	ID             string    `json:"id"`
	Purpose        string    `json:"purpose"`
	RetentionClass string    `json:"retention_class"`
	RepoID         *string   `json:"repo_id,omitempty"`
	Branch         *string   `json:"branch,omitempty"`
	CreatedBy      *string   `json:"created_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TraceListFilter represents filters for listing traces
type TraceListFilter struct {
	RepoID         *string `json:"repo_id,omitempty" form:"repo_id" binding:"omitempty,uuid4"`
	Purpose        *string `json:"purpose,omitempty" form:"purpose" binding:"omitempty,oneof=release deployment build test"`
	RetentionClass *string `json:"retention_class,omitempty" form:"retention_class" binding:"omitempty,oneof=short long permanent"`
	Branch         *string `json:"branch,omitempty" form:"branch"`
	CreatedBy      *string `json:"created_by,omitempty" form:"created_by"`
	TimeRange
	Pagination
	Sort
}

// TraceIDParam represents a trace ID parameter
type TraceIDParam struct {
	TraceID uuid.UUID `uri:"trace_id" binding:"required,uuid4"`
}
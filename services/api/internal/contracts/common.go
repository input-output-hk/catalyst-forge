package contracts

import (
	"time"

	"github.com/google/uuid"
)

// Pagination represents pagination request parameters
type Pagination struct {
	Page     int `json:"page" form:"page" binding:"min=1"`
	PageSize int `json:"page_size" form:"page_size" binding:"min=1,max=100"`
}

// PageResult represents a paginated response
type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// Specific PageResult types for Swagger compatibility
// These are needed because swag doesn't handle generics well

// ArtifactPageResult represents a paginated list of artifacts
type ArtifactPageResult struct {
	Items    []ArtifactResponse `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int64              `json:"total"`
}

// BuildPageResult represents a paginated list of builds
type BuildPageResult struct {
	Items    []BuildResponse `json:"items"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int64           `json:"total"`
}

// DeploymentPageResult represents a paginated list of deployments
type DeploymentPageResult struct {
	Items    []DeploymentResponse `json:"items"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	Total    int64                `json:"total"`
}

// EnvironmentPageResult represents a paginated list of environments
type EnvironmentPageResult struct {
	Items    []EnvironmentResponse `json:"items"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	Total    int64                 `json:"total"`
}

// ProjectPageResult represents a paginated list of projects
type ProjectPageResult struct {
	Items    []ProjectResponse `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Total    int64             `json:"total"`
}

// ReleasePageResult represents a paginated list of releases
type ReleasePageResult struct {
	Items    []ReleaseResponse `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Total    int64             `json:"total"`
}

// RenderedReleasePageResult represents a paginated list of rendered releases
type RenderedReleasePageResult struct {
	Items    []RenderedReleaseResponse `json:"items"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
	Total    int64                     `json:"total"`
}

// RepositoryPageResult represents a paginated list of repositories
type RepositoryPageResult struct {
	Items    []RepositoryResponse `json:"items"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	Total    int64                `json:"total"`
}

// TracePageResult represents a paginated list of traces
type TracePageResult struct {
	Items    []TraceResponse `json:"items"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Total    int64           `json:"total"`
}

// Sort represents sorting parameters
type Sort struct {
	Field string `json:"field" form:"sort_field"`
	Order string `json:"order" form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// TimeRange represents a time range filter
type TimeRange struct {
	Since *time.Time `json:"since,omitempty" form:"since"`
	Until *time.Time `json:"until,omitempty" form:"until"`
}

// UUIDParam represents a UUID parameter
type UUIDParam struct {
	ID uuid.UUID `uri:"id" binding:"required,uuid4"`
}

// StringParam represents a string parameter
type StringParam struct {
	Value string `uri:"value" binding:"required"`
}

// NewPageResult creates a new page result
func NewPageResult[T any](items []T, page, pageSize int, total int64) PageResult[T] {
	if items == nil {
		items = []T{}
	}
	return PageResult[T]{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}
}

// NewErrorResponse creates a new error response
func NewErrorResponse(code, message string, details interface{}) ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// Common error codes
const (
	ErrCodeBadRequest          = "bad_request"
	ErrCodeUnauthorized        = "unauthorized"
	ErrCodeForbidden           = "forbidden"
	ErrCodeNotFound            = "not_found"
	ErrCodeConflict            = "conflict"
	ErrCodeUnprocessableEntity = "unprocessable_entity"
	ErrCodeInternalError       = "internal_error"
	ErrCodeServiceUnavailable  = "service_unavailable"
)

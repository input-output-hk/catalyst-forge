package v2

import (
	"time"

	"github.com/google/uuid"
)

// DeploymentCreate represents a request to create a deployment
type DeploymentCreate struct {
	ReleaseID     string  `json:"release_id" binding:"required,uuid4"`
	EnvironmentID string  `json:"environment_id" binding:"required,uuid4"`
	Status        *string `json:"status,omitempty" binding:"omitempty,oneof=pending submitted failed succeeded canceled"`
	IntentDigest  *string `json:"intent_digest,omitempty"`
	StatusReason  *string `json:"status_reason,omitempty"`
	DeployedBy    *string `json:"deployed_by,omitempty"`
}

// DeploymentUpdate represents a request to update a deployment
type DeploymentUpdate struct {
	Status       *string    `json:"status,omitempty" binding:"omitempty,oneof=pending submitted failed succeeded canceled"`
	StatusReason *string    `json:"status_reason,omitempty"`
	DeployedAt   *time.Time `json:"deployed_at,omitempty"`
}

// DeploymentResponse represents a deployment response
type DeploymentResponse struct {
	ID            string     `json:"id"`
	ReleaseID     string     `json:"release_id"`
	EnvironmentID string     `json:"environment_id"`
	Status        string     `json:"status"`
	IntentDigest  *string    `json:"intent_digest,omitempty"`
	StatusReason  *string    `json:"status_reason,omitempty"`
	DeployedBy    *string    `json:"deployed_by,omitempty"`
	DeployedAt    *time.Time `json:"deployed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// DeploymentListFilter represents filters for listing deployments
type DeploymentListFilter struct {
	ReleaseID     *string `json:"release_id,omitempty" form:"release_id" binding:"omitempty,uuid4"`
	EnvironmentID *string `json:"environment_id,omitempty" form:"environment_id" binding:"omitempty,uuid4"`
	Status        *string `json:"status,omitempty" form:"status" binding:"omitempty,oneof=pending submitted failed succeeded canceled"`
	DeployedBy    *string `json:"deployed_by,omitempty" form:"deployed_by"`
	TimeRange
	Pagination
	Sort
}

// DeploymentIDParam represents a deployment ID parameter
type DeploymentIDParam struct {
	DeploymentID uuid.UUID `uri:"deployment_id" binding:"required,uuid4"`
}

// RenderJobCreate represents a request to create a render job
type RenderJobCreate struct {
	DeploymentID       string                 `json:"deployment_id" binding:"required,uuid4"`
	Status             *string                `json:"status,omitempty" binding:"omitempty,oneof=pending running succeeded failed"`
	Template           map[string]interface{} `json:"template,omitempty"`
	Rendered           map[string]interface{} `json:"rendered,omitempty"`
	ValidationMessages []string               `json:"validation_messages,omitempty"`
	RenderError        *string                `json:"render_error,omitempty"`
}

// RenderJobUpdate represents a request to update a render job
type RenderJobUpdate struct {
	Status             *string                `json:"status,omitempty" binding:"omitempty,oneof=pending running succeeded failed"`
	Rendered           map[string]interface{} `json:"rendered,omitempty"`
	ValidationMessages []string               `json:"validation_messages,omitempty"`
	RenderError        *string                `json:"render_error,omitempty"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
}

// RenderJobResponse represents a render job response
type RenderJobResponse struct {
	ID                 string                 `json:"id"`
	DeploymentID       string                 `json:"deployment_id"`
	Status             string                 `json:"status"`
	Template           map[string]interface{} `json:"template,omitempty"`
	Rendered           map[string]interface{} `json:"rendered,omitempty"`
	ValidationMessages []string               `json:"validation_messages,omitempty"`
	RenderError        *string                `json:"render_error,omitempty"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// RenderJobListFilter represents filters for listing render jobs
type RenderJobListFilter struct {
	DeploymentID *string `json:"deployment_id,omitempty" form:"deployment_id" binding:"omitempty,uuid4"`
	Status       *string `json:"status,omitempty" form:"status" binding:"omitempty,oneof=pending running succeeded failed"`
	TimeRange
	Pagination
	Sort
}

// RenderJobIDParam represents a render job ID parameter
type RenderJobIDParam struct {
	RenderJobID uuid.UUID `uri:"render_job_id" binding:"required,uuid4"`
}
package contracts

import (
	"time"
)

// DeploymentCreate represents a request to create a deployment
type DeploymentCreate struct {
	ReleaseID     string  `json:"release_id" binding:"required,uuid4"`
	EnvironmentID string  `json:"environment_id" binding:"required,uuid4"`
	Status        *string `json:"status,omitempty" binding:"omitempty,oneof=pending rendered pushed reconciling healthy degraded failed rolled_back"`
	IntentDigest  *string `json:"intent_digest,omitempty"`
	StatusReason  *string `json:"status_reason,omitempty"`
	DeployedBy    *string `json:"deployed_by,omitempty"`
}

// DeploymentUpdate represents a request to update a deployment
type DeploymentUpdate struct {
	Status       *string    `json:"status,omitempty" binding:"omitempty,oneof=pending rendered pushed reconciling healthy degraded failed rolled_back"`
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
	Status        *string `json:"status,omitempty" form:"status" binding:"omitempty,oneof=pending rendered pushed reconciling healthy degraded failed rolled_back"`
	DeployedBy    *string `json:"deployed_by,omitempty" form:"deployed_by"`
	TimeRange
	Pagination
	Sort
}

// DeploymentIDParam represents a deployment ID parameter
type DeploymentIDParam struct {
	DeploymentID string `uri:"deployment_id" binding:"required"`
}

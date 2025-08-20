package contracts

import (
	"time"

	"github.com/google/uuid"
)

// GitOpsChangeCreate represents a request to create a GitOps change
type GitOpsChangeCreate struct {
	DeploymentID   string                 `json:"deployment_id" binding:"required,uuid4"`
	ChangeType     string                 `json:"change_type" binding:"required,oneof=create update delete"`
	ManifestBefore map[string]interface{} `json:"manifest_before,omitempty"`
	ManifestAfter  map[string]interface{} `json:"manifest_after,omitempty"`
	FilePath       string                 `json:"file_path" binding:"required"`
	CommitSHA      *string                `json:"commit_sha,omitempty"`
	PullRequestID  *string                `json:"pull_request_id,omitempty"`
	Branch         *string                `json:"branch,omitempty"`
	Applied        bool                   `json:"applied"`
	AppliedBy      *string                `json:"applied_by,omitempty"`
	AppliedAt      *time.Time             `json:"applied_at,omitempty"`
}

// GitOpsChangeUpdate represents a request to update a GitOps change
type GitOpsChangeUpdate struct {
	CommitSHA     *string    `json:"commit_sha,omitempty"`
	PullRequestID *string    `json:"pull_request_id,omitempty"`
	Branch        *string    `json:"branch,omitempty"`
	Applied       *bool      `json:"applied,omitempty"`
	AppliedBy     *string    `json:"applied_by,omitempty"`
	AppliedAt     *time.Time `json:"applied_at,omitempty"`
}

// GitOpsChangeResponse represents a GitOps change response
type GitOpsChangeResponse struct {
	ID             string                 `json:"id"`
	DeploymentID   string                 `json:"deployment_id"`
	ChangeType     string                 `json:"change_type"`
	ManifestBefore map[string]interface{} `json:"manifest_before,omitempty"`
	ManifestAfter  map[string]interface{} `json:"manifest_after,omitempty"`
	FilePath       string                 `json:"file_path"`
	CommitSHA      *string                `json:"commit_sha,omitempty"`
	PullRequestID  *string                `json:"pull_request_id,omitempty"`
	Branch         *string                `json:"branch,omitempty"`
	Applied        bool                   `json:"applied"`
	AppliedBy      *string                `json:"applied_by,omitempty"`
	AppliedAt      *time.Time             `json:"applied_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// GitOpsChangeListFilter represents filters for listing GitOps changes
type GitOpsChangeListFilter struct {
	DeploymentID  *string `json:"deployment_id,omitempty" form:"deployment_id" binding:"omitempty,uuid4"`
	ChangeType    *string `json:"change_type,omitempty" form:"change_type" binding:"omitempty,oneof=create update delete"`
	FilePath      *string `json:"file_path,omitempty" form:"file_path"`
	CommitSHA     *string `json:"commit_sha,omitempty" form:"commit_sha"`
	PullRequestID *string `json:"pull_request_id,omitempty" form:"pull_request_id"`
	Branch        *string `json:"branch,omitempty" form:"branch"`
	Applied       *bool   `json:"applied,omitempty" form:"applied"`
	AppliedBy     *string `json:"applied_by,omitempty" form:"applied_by"`
	TimeRange
	Pagination
	Sort
}

// GitOpsChangeIDParam represents a GitOps change ID parameter
type GitOpsChangeIDParam struct {
	GitOpsChangeID uuid.UUID `uri:"gitops_change_id" binding:"required,uuid4"`
}

// GitOpsSyncCreate represents a request to create a GitOps sync status
type GitOpsSyncCreate struct {
	DeploymentID   string     `json:"deployment_id" binding:"required,uuid4"`
	AppName        string     `json:"app_name" binding:"required"`
	AppNamespace   string     `json:"app_namespace" binding:"required"`
	SyncStatus     string     `json:"sync_status" binding:"required,oneof=synced out_of_sync unknown"`
	HealthStatus   string     `json:"health_status" binding:"required,oneof=healthy progressing degraded suspended missing unknown"`
	Revision       string     `json:"revision" binding:"required"`
	Message        *string    `json:"message,omitempty"`
	SyncStartedAt  *time.Time `json:"sync_started_at,omitempty"`
	SyncFinishedAt *time.Time `json:"sync_finished_at,omitempty"`
}

// GitOpsSyncUpdate represents a request to update a GitOps sync status
type GitOpsSyncUpdate struct {
	SyncStatus     *string    `json:"sync_status,omitempty" binding:"omitempty,oneof=synced out_of_sync unknown"`
	HealthStatus   *string    `json:"health_status,omitempty" binding:"omitempty,oneof=healthy progressing degraded suspended missing unknown"`
	Revision       *string    `json:"revision,omitempty"`
	Message        *string    `json:"message,omitempty"`
	SyncStartedAt  *time.Time `json:"sync_started_at,omitempty"`
	SyncFinishedAt *time.Time `json:"sync_finished_at,omitempty"`
}

// GitOpsSyncResponse represents a GitOps sync status response
type GitOpsSyncResponse struct {
	ID             string     `json:"id"`
	DeploymentID   string     `json:"deployment_id"`
	AppName        string     `json:"app_name"`
	AppNamespace   string     `json:"app_namespace"`
	SyncStatus     string     `json:"sync_status"`
	HealthStatus   string     `json:"health_status"`
	Revision       string     `json:"revision"`
	Message        *string    `json:"message,omitempty"`
	SyncStartedAt  *time.Time `json:"sync_started_at,omitempty"`
	SyncFinishedAt *time.Time `json:"sync_finished_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// GitOpsSyncListFilter represents filters for listing GitOps sync statuses
type GitOpsSyncListFilter struct {
	DeploymentID *string `json:"deployment_id,omitempty" form:"deployment_id" binding:"omitempty,uuid4"`
	AppName      *string `json:"app_name,omitempty" form:"app_name"`
	AppNamespace *string `json:"app_namespace,omitempty" form:"app_namespace"`
	SyncStatus   *string `json:"sync_status,omitempty" form:"sync_status" binding:"omitempty,oneof=synced out_of_sync unknown"`
	HealthStatus *string `json:"health_status,omitempty" form:"health_status" binding:"omitempty,oneof=healthy progressing degraded suspended missing unknown"`
	TimeRange
	Pagination
	Sort
}

// GitOpsSyncIDParam represents a GitOps sync ID parameter
type GitOpsSyncIDParam struct {
	GitOpsSyncID uuid.UUID `uri:"gitops_sync_id" binding:"required,uuid4"`
}

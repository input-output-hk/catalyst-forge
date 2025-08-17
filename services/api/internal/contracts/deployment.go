package contracts

import (
	"time"

	"github.com/google/uuid"
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

// ModuleVersion represents a module version in render job
type ModuleVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// RenderJobCreate represents a request to create a render job
type RenderJobCreate struct {
	ModuleVersions  []ModuleVersion `json:"module_versions,omitempty"`
	BundleHash      *string         `json:"bundle_hash,omitempty"`
	RendererVersion *string         `json:"renderer_version,omitempty"`
}

// RenderJobUpdate represents a request to update a render job
type RenderJobUpdate struct {
	Status              *string         `json:"status,omitempty" binding:"omitempty,oneof=pending running succeeded failed"`
	ModuleVersions      []ModuleVersion `json:"module_versions,omitempty"`
	BundleHash          *string         `json:"bundle_hash,omitempty"`
	OutputHash          *string         `json:"output_hash,omitempty"`
	StorageURI          *string         `json:"storage_uri,omitempty"`
	RendererVersion     *string         `json:"renderer_version,omitempty"`
	OCIRef              *string         `json:"oci_ref,omitempty"`
	OCIDigest           *string         `json:"oci_digest,omitempty"`
	Signed              *bool           `json:"signed,omitempty"`
	SignatureVerifiedAt *time.Time      `json:"signature_verified_at,omitempty"`
	FinishedAt          *time.Time      `json:"finished_at,omitempty"`
}

// RenderJobResponse represents a render job response
type RenderJobResponse struct {
	ID                  string          `json:"id"`
	DeploymentID        string          `json:"deployment_id"`
	Status              string          `json:"status"`
	ModuleVersions      []ModuleVersion `json:"module_versions,omitempty"`
	BundleHash          *string         `json:"bundle_hash,omitempty"`
	OutputHash          *string         `json:"output_hash,omitempty"`
	StorageURI          *string         `json:"storage_uri,omitempty"`
	RendererVersion     *string         `json:"renderer_version,omitempty"`
	OCIRef              *string         `json:"oci_ref,omitempty"`
	OCIDigest           *string         `json:"oci_digest,omitempty"`
	Signed              bool            `json:"signed"`
	SignatureVerifiedAt *time.Time      `json:"signature_verified_at,omitempty"`
	StartedAt           time.Time       `json:"started_at"`
	FinishedAt          *time.Time      `json:"finished_at,omitempty"`
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

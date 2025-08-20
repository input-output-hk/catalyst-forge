package contracts

import (
	"time"

	"github.com/google/uuid"
)

// RenderedReleaseCreate represents a request to create a rendered release record
type RenderedReleaseCreate struct {
	DeploymentID        string                   `json:"deployment_id" binding:"required,uuid4"`
	ReleaseID           string                   `json:"release_id" binding:"required,uuid4"`
	EnvironmentID       string                   `json:"environment_id" binding:"required,uuid4"`
	RendererVersion     string                   `json:"renderer_version" binding:"required"`
	ModuleVersions      []map[string]interface{} `json:"module_versions"`
	BundleHash          string                   `json:"bundle_hash" binding:"required"`
	OutputHash          string                   `json:"output_hash" binding:"required"`
	OCIRef              string                   `json:"oci_ref" binding:"required"`
	OCIDigest           string                   `json:"oci_digest" binding:"required"`
	StorageURI          *string                  `json:"storage_uri,omitempty"`
	Signed              *bool                    `json:"signed,omitempty"`
	SignatureVerifiedAt *time.Time               `json:"signature_verified_at,omitempty"`
}

// RenderedReleaseUpdate represents a request to update a rendered release
type RenderedReleaseUpdate struct {
	OCIRef              *string    `json:"oci_ref,omitempty"`
	OCIDigest           *string    `json:"oci_digest,omitempty"`
	StorageURI          *string    `json:"storage_uri,omitempty"`
	Signed              *bool      `json:"signed,omitempty"`
	SignatureVerifiedAt *time.Time `json:"signature_verified_at,omitempty"`
}

// RenderedReleaseResponse represents a rendered release response
type RenderedReleaseResponse struct {
	ID                  string                   `json:"id"`
	DeploymentID        string                   `json:"deployment_id"`
	ReleaseID           string                   `json:"release_id"`
	EnvironmentID       string                   `json:"environment_id"`
	RendererVersion     string                   `json:"renderer_version"`
	ModuleVersions      []map[string]interface{} `json:"module_versions"`
	BundleHash          string                   `json:"bundle_hash"`
	OutputHash          string                   `json:"output_hash"`
	OCIRef              string                   `json:"oci_ref"`
	OCIDigest           string                   `json:"oci_digest"`
	StorageURI          *string                  `json:"storage_uri,omitempty"`
	Signed              bool                     `json:"signed"`
	SignatureVerifiedAt *time.Time               `json:"signature_verified_at,omitempty"`
	CreatedAt           time.Time                `json:"created_at"`
	UpdatedAt           time.Time                `json:"updated_at"`
}

// RenderedReleaseListFilter represents filters for listing rendered releases
type RenderedReleaseListFilter struct {
	ReleaseID     *string `json:"release_id,omitempty" form:"release_id" binding:"omitempty,uuid4"`
	EnvironmentID *string `json:"environment_id,omitempty" form:"environment_id" binding:"omitempty,uuid4"`
	DeploymentID  *string `json:"deployment_id,omitempty" form:"deployment_id" binding:"omitempty,uuid4"`
	OCIDigest     *string `json:"oci_digest,omitempty" form:"oci_digest"`
	OutputHash    *string `json:"output_hash,omitempty" form:"output_hash"`
	TimeRange
	Pagination
	Sort
}

// RenderedReleaseIDParam represents a rendered release ID parameter
type RenderedReleaseIDParam struct {
	RenderedReleaseID uuid.UUID `uri:"rendered_release_id" binding:"required,uuid4"`
}

// RenderedReleaseDeploymentParam represents a deployment ID parameter
type RenderedReleaseDeploymentParam struct {
	DeploymentID uuid.UUID `uri:"deployment_id" binding:"required,uuid4"`
}

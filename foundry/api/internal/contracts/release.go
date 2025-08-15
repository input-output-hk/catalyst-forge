package contracts

import (
	"time"

	"github.com/google/uuid"
)

// ReleaseCreate represents a request to create a release
type ReleaseCreate struct {
	ProjectID      string                 `json:"project_id" binding:"required,uuid4"`
	ReleaseKey     string                 `json:"release_key" binding:"required"`
	TraceID        *string                `json:"trace_id,omitempty" binding:"omitempty,uuid4"`
	SourceCommit   string                 `json:"source_commit" binding:"required"`
	SourceBranch   *string                `json:"source_branch,omitempty"`
	Tag            *string                `json:"tag,omitempty"`
	Status         *string                `json:"status,omitempty" binding:"omitempty,oneof=draft sealed"`
	OCIRef         *string                `json:"oci_ref,omitempty"`
	OCIDigest      *string                `json:"oci_digest,omitempty"`
	ValuesHash     *string                `json:"values_hash,omitempty"`
	ValuesSnapshot map[string]interface{} `json:"values_snapshot,omitempty"`
	ContentHash    *string                `json:"content_hash,omitempty"`
	CreatedBy      *string                `json:"created_by,omitempty"`
	Modules        []ReleaseModule        `json:"modules,omitempty"`
	Injections     []ReleaseInjection     `json:"injections,omitempty"`
	Artifacts      []ReleaseArtifactLink  `json:"artifacts,omitempty"`
}

// ReleaseUpdate represents a request to update a release
type ReleaseUpdate struct {
	Status              *string    `json:"status,omitempty" binding:"omitempty,oneof=draft sealed"`
	OCIRef              *string    `json:"oci_ref,omitempty"`
	OCIDigest           *string    `json:"oci_digest,omitempty"`
	Signed              *bool      `json:"signed,omitempty"`
	SigIssuer           *string    `json:"sig_issuer,omitempty"`
	SigSubject          *string    `json:"sig_subject,omitempty"`
	SignatureVerifiedAt *time.Time `json:"signature_verified_at,omitempty"`
}

// ReleaseResponse represents a release response
type ReleaseResponse struct {
	ID                  string                 `json:"id"`
	ProjectID           string                 `json:"project_id"`
	ReleaseKey          string                 `json:"release_key"`
	TraceID             *string                `json:"trace_id,omitempty"`
	SourceCommit        string                 `json:"source_commit"`
	SourceBranch        *string                `json:"source_branch,omitempty"`
	Tag                 *string                `json:"tag,omitempty"`
	Status              string                 `json:"status"`
	OCIRef              *string                `json:"oci_ref,omitempty"`
	OCIDigest           *string                `json:"oci_digest,omitempty"`
	Signed              bool                   `json:"signed"`
	SigIssuer           *string                `json:"sig_issuer,omitempty"`
	SigSubject          *string                `json:"sig_subject,omitempty"`
	SignatureVerifiedAt *time.Time             `json:"signature_verified_at,omitempty"`
	ValuesHash          *string                `json:"values_hash,omitempty"`
	ValuesSnapshot      map[string]interface{} `json:"values_snapshot,omitempty"`
	ContentHash         *string                `json:"content_hash,omitempty"`
	CreatedBy           *string                `json:"created_by,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// ReleaseListFilter represents filters for listing releases
type ReleaseListFilter struct {
	ProjectID  *string `json:"project_id,omitempty" form:"project_id" binding:"omitempty,uuid4"`
	ReleaseKey *string `json:"release_key,omitempty" form:"release_key"`
	Status     *string `json:"status,omitempty" form:"status" binding:"omitempty,oneof=draft sealed"`
	OCIDigest  *string `json:"oci_digest,omitempty" form:"oci_digest"`
	Tag        *string `json:"tag,omitempty" form:"tag"`
	CreatedBy  *string `json:"created_by,omitempty" form:"created_by"`
	TimeRange
	Pagination
	Sort
}

// ReleaseModule represents a release module
type ReleaseModule struct {
	ID         string  `json:"id,omitempty"`
	ReleaseID  string  `json:"release_id,omitempty"`
	ModuleKey  string  `json:"module_key" binding:"required"`
	Name       string  `json:"name" binding:"required"`
	ModuleType string  `json:"module_type" binding:"required,oneof=kcl helm git"`
	Version    *string `json:"version,omitempty"`
	Registry   *string `json:"registry,omitempty"`
	OCIRef     *string `json:"oci_ref,omitempty"`
	OCIDigest  *string `json:"oci_digest,omitempty"`
	GitURL     *string `json:"git_url,omitempty"`
	GitRef     *string `json:"git_ref,omitempty"`
	Path       *string `json:"path,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
}

// ReleaseModuleCreate represents a request to create release modules
type ReleaseModuleCreate struct {
	Modules []ReleaseModule `json:"modules" binding:"required,min=1,dive"`
}

// ReleaseModuleUpdate represents a request to update a release module
type ReleaseModuleUpdate struct {
	Name       string  `json:"name" binding:"required"`
	ModuleType string  `json:"module_type" binding:"required,oneof=kcl helm git"`
	Version    *string `json:"version,omitempty"`
	Registry   *string `json:"registry,omitempty"`
	OCIRef     *string `json:"oci_ref,omitempty"`
	OCIDigest  *string `json:"oci_digest,omitempty"`
	GitURL     *string `json:"git_url,omitempty"`
	GitRef     *string `json:"git_ref,omitempty"`
	Path       *string `json:"path,omitempty"`
}

// ReleaseInjection represents a release injection
type ReleaseInjection struct {
	ID            string     `json:"id,omitempty"`
	ReleaseID     string     `json:"release_id,omitempty"`
	JSONPointer   string     `json:"json_pointer" binding:"required"`
	ArtifactKey   string     `json:"artifact_key" binding:"required"`
	ArtifactField string     `json:"artifact_field" binding:"required,oneof=image_name image_digest tag repo"`
	ModuleKey     *string    `json:"module_key,omitempty"`
	ModuleName    *string    `json:"module_name,omitempty"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
}

// ReleaseInjectionCreate represents a request to create release injections
type ReleaseInjectionCreate struct {
	Injections []ReleaseInjection `json:"injections" binding:"required,min=1,dive"`
}

// ReleaseArtifactLink represents a link between a release and an artifact
type ReleaseArtifactLink struct {
	ArtifactID  string  `json:"artifact_id" binding:"required,uuid4"`
	Role        string  `json:"role" binding:"required"`
	ArtifactKey *string `json:"artifact_key,omitempty"`
}

// ReleaseArtifactResponse represents a release artifact response
type ReleaseArtifactResponse struct {
	ReleaseID   string    `json:"release_id"`
	ArtifactID  string    `json:"artifact_id"`
	Role        string    `json:"role"`
	ArtifactKey *string   `json:"artifact_key,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ReleaseArtifactCreate represents a request to attach an artifact to a release
type ReleaseArtifactCreate struct {
	ArtifactID  string  `json:"artifact_id" binding:"required,uuid4"`
	Role        string  `json:"role" binding:"required"`
	ArtifactKey *string `json:"artifact_key,omitempty"`
}

// ReleaseIDParam represents a release ID parameter
type ReleaseIDParam struct {
	ReleaseID uuid.UUID `uri:"release_id" binding:"required,uuid4"`
}

// ReleaseModuleKeyParam represents a module key parameter
type ReleaseModuleKeyParam struct {
	ReleaseID  uuid.UUID `uri:"release_id" binding:"required,uuid4"`
	ModuleKey  string    `uri:"module_key" binding:"required"`
}

// ReleaseInjectionIDParam represents an injection ID parameter
type ReleaseInjectionIDParam struct {
	ReleaseID    uuid.UUID `uri:"release_id" binding:"required,uuid4"`
	InjectionID  uuid.UUID `uri:"injection_id" binding:"required,uuid4"`
}

// ReleaseArtifactIDParam represents an artifact ID parameter for release
type ReleaseArtifactIDParam struct {
	ReleaseID   uuid.UUID `uri:"release_id" binding:"required,uuid4"`
	ArtifactID  uuid.UUID `uri:"artifact_id" binding:"required,uuid4"`
	Role        string    `form:"role" binding:"required"`
}
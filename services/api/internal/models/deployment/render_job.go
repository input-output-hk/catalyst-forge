package deployment

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/enums"
)

// RenderJob represents renderer execution for a Deployment (can publish a Rendered Set OCI)
type RenderJob struct {
	ID                  uuid.UUID             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeploymentID        uuid.UUID             `gorm:"type:uuid;not null;uniqueIndex" json:"deployment_id"`
	ModuleVersions      JSONB                 `gorm:"type:jsonb" json:"module_versions,omitempty"` // [{"name":"app","version":"0.5.0"}]
	BundleHash          *string               `json:"bundle_hash,omitempty"`
	OutputHash          *string               `json:"output_hash,omitempty"`
	StorageURI          *string               `json:"storage_uri,omitempty"`
	Status              enums.RenderJobStatus `gorm:"not null" json:"status"`
	RendererVersion     *string               `json:"renderer_version,omitempty"`
	OCIRef              *string               `json:"oci_ref,omitempty"`
	OCIDigest           *string               `gorm:"column:oci_digest;index:ix_render_job_oci_digest,where:oci_digest IS NOT NULL" json:"oci_digest,omitempty"`
	Signed              bool                  `gorm:"not null;default:false" json:"signed"`
	SignatureVerifiedAt *time.Time            `json:"signature_verified_at,omitempty"`
	StartedAt           time.Time             `gorm:"not null;default:now()" json:"started_at"`
	FinishedAt          *time.Time            `json:"finished_at,omitempty"`
}

// TableName specifies the table name
func (RenderJob) TableName() string {
	return "render_job"
}

// BeforeCreate hook to set UUID if not provided
func (rj *RenderJob) BeforeCreate(tx *gorm.DB) error {
	if rj.ID == uuid.Nil {
		rj.ID = uuid.New()
	}
	return nil
}

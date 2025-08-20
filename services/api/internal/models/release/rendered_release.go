package release

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RenderedRelease represents a rendered release bundle and its immutable outputs
type RenderedRelease struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	DeploymentID  uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"deployment_id"`
	ReleaseID     uuid.UUID `gorm:"type:uuid;index;not null" json:"release_id"`
	EnvironmentID uuid.UUID `gorm:"type:uuid;index;not null" json:"environment_id"`

	RendererVersion string         `gorm:"not null" json:"renderer_version"`
	ModuleVersions  datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"module_versions"`
	BundleHash      string         `gorm:"not null" json:"bundle_hash"`
	OutputHash      string         `gorm:"index;not null" json:"output_hash"`

	OCIRef     string  `gorm:"not null" json:"oci_ref"`
	OCIDigest  string  `gorm:"index;not null" json:"oci_digest"`
	StorageURI *string `json:"storage_uri,omitempty"`

	Signed              bool       `gorm:"not null;default:false" json:"signed"`
	SignatureVerifiedAt *time.Time `json:"signature_verified_at,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name
func (RenderedRelease) TableName() string {
	return "rendered_release"
}

// BeforeCreate hook to set UUID if not provided
func (r *RenderedRelease) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

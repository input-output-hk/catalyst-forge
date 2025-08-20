package release

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/enums"
)

// JSONB handles JSONB field type
type JSONB map[string]any

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Release represents an immutable binding of commit + selected artifacts + module lock + base values integrity
type Release struct {
	ID                  uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProjectID           uuid.UUID           `gorm:"type:uuid;not null" json:"project_id"`
	ReleaseKey          string              `gorm:"not null;uniqueIndex:ux_release_project_key,priority:2" json:"release_key"`
	Alias               *string             `json:"alias,omitempty"`
	Semver              *string             `json:"semver,omitempty"`
	TraceID             *uuid.UUID          `gorm:"type:uuid" json:"trace_id,omitempty"`
	SourceCommit        string              `gorm:"not null" json:"source_commit"`
	SourceBranch        *string             `json:"source_branch,omitempty"`
	Tag                 *string             `gorm:"uniqueIndex:ux_release_project_tag,priority:2,where:tag IS NOT NULL" json:"tag,omitempty"`
	Status              enums.ReleaseStatus `gorm:"not null;default:'draft'" json:"status"`
	OCIRef              *string             `json:"oci_ref,omitempty"`
	OCIDigest           *string             `gorm:"column:oci_digest;uniqueIndex:ux_release_oci_digest,where:oci_digest IS NOT NULL" json:"oci_digest,omitempty"`
	Signed              bool                `gorm:"not null;default:false" json:"signed"`
	SigIssuer           *string             `json:"sig_issuer,omitempty"`
	SigSubject          *string             `json:"sig_subject,omitempty"`
	SignatureVerifiedAt *time.Time          `json:"signature_verified_at,omitempty"`
	ValuesHash          *string             `json:"values_hash,omitempty"`
	ValuesSnapshot      JSONB               `gorm:"type:jsonb" json:"values_snapshot,omitempty"` // optional base values snapshot (no env overlay)
	ContentHash         *string             `json:"content_hash,omitempty"`
	CreatedBy           *string             `json:"created_by,omitempty"`
	CreatedAt           time.Time           `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt           time.Time           `gorm:"not null;default:now()" json:"updated_at"`

	// Relationships
	Modules []ReleaseModule `gorm:"foreignKey:ReleaseID" json:"modules,omitempty"`
	// Injections removed in v2
	Artifacts []ReleaseArtifact `gorm:"foreignKey:ReleaseID" json:"artifacts,omitempty"`
}

// TableName specifies the table name
func (Release) TableName() string {
	return "release"
}

// BeforeCreate hook to set UUID if not provided
func (r *Release) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// BeforeUpdate hook to enforce immutability when sealed
func (r *Release) BeforeUpdate(tx *gorm.DB) error {
	if r.Status == enums.ReleaseStatusSealed {
		// Check if trying to update immutable fields
		var oldRelease Release
		if err := tx.Model(&Release{}).Where("id = ?", r.ID).First(&oldRelease).Error; err != nil {
			return err
		}

		if oldRelease.Status == enums.ReleaseStatusSealed {
			// These fields must not change once sealed
			if oldRelease.SourceCommit != r.SourceCommit ||
				(oldRelease.Tag != nil && r.Tag != nil && *oldRelease.Tag != *r.Tag) ||
				(oldRelease.ContentHash != nil && r.ContentHash != nil && *oldRelease.ContentHash != *r.ContentHash) ||
				(oldRelease.OCIDigest != nil && r.OCIDigest != nil && *oldRelease.OCIDigest != *r.OCIDigest) ||
				(oldRelease.ValuesHash != nil && r.ValuesHash != nil && *oldRelease.ValuesHash != *r.ValuesHash) {
				return errors.New("cannot modify immutable fields on sealed release")
			}
		}
	}
	return nil
}

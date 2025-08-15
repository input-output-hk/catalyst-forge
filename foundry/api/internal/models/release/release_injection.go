package release

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ArtifactField represents the artifact field type
type ArtifactField string

const (
	ArtifactFieldImageName   ArtifactField = "image_name"
	ArtifactFieldImageDigest ArtifactField = "image_digest"
	ArtifactFieldTag         ArtifactField = "tag"
	ArtifactFieldRepo        ArtifactField = "repo"
)

// ReleaseInjection represents injection plan extracted at release time from @artifact attributes
type ReleaseInjection struct {
	ID            uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ReleaseID     uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:ux_release_injection_path,priority:1" json:"release_id"`
	ModuleKey     *string       `json:"module_key,omitempty"`
	ModuleName    *string       `json:"module_name,omitempty"`
	JSONPointer   string        `gorm:"not null;uniqueIndex:ux_release_injection_path,priority:2" json:"json_pointer"` // RFC6901 JSON Pointer into base values
	ArtifactKey   string        `gorm:"not null" json:"artifact_key"`
	ArtifactField ArtifactField `gorm:"not null" json:"artifact_field"`
	CreatedAt     time.Time     `gorm:"not null;default:now()" json:"created_at"`
}

// TableName specifies the table name
func (ReleaseInjection) TableName() string {
	return "release_injection"
}

// BeforeCreate hook to set UUID if not provided
func (ri *ReleaseInjection) BeforeCreate(tx *gorm.DB) error {
	if ri.ID == uuid.Nil {
		ri.ID = uuid.New()
	}
	return nil
}
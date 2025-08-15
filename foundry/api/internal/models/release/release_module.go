package release

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModuleType represents the type of module
type ModuleType string

const (
	ModuleTypeKCL  ModuleType = "kcl"
	ModuleTypeHelm ModuleType = "helm"
	ModuleTypeGit  ModuleType = "git"
)

// ReleaseModule represents module lock per Release (exact sources used)
type ReleaseModule struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ReleaseID  uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:ux_release_module_key,priority:1" json:"release_id"`
	ModuleKey  string     `gorm:"not null;uniqueIndex:ux_release_module_key,priority:2" json:"module_key"` // key in bundle (e.g., 'main','crd')
	Name       string     `gorm:"not null" json:"name"`
	ModuleType ModuleType `gorm:"not null" json:"module_type"`
	Version    *string    `json:"version,omitempty"`
	Registry   *string    `json:"registry,omitempty"`
	OCIRef     *string    `json:"oci_ref,omitempty"`
	OCIDigest  *string    `json:"oci_digest,omitempty"`
	GitURL     *string    `json:"git_url,omitempty"`
	GitRef     *string    `json:"git_ref,omitempty"`
	Path       *string    `json:"path,omitempty"`
	CreatedAt  time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

// TableName specifies the table name
func (ReleaseModule) TableName() string {
	return "release_module"
}

// BeforeCreate hook to set UUID if not provided
func (rm *ReleaseModule) BeforeCreate(tx *gorm.DB) error {
	if rm.ID == uuid.Nil {
		rm.ID = uuid.New()
	}
	return nil
}
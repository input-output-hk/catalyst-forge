package release

import (
	"time"

	"github.com/google/uuid"
)

// ReleaseArtifact represents selected build artifacts included in a Release
type ReleaseArtifact struct {
	ReleaseID   uuid.UUID `gorm:"type:uuid;not null;primaryKey;uniqueIndex:ux_release_artifact_key,priority:1,where:artifact_key IS NOT NULL" json:"release_id"`
	ArtifactID  uuid.UUID `gorm:"type:uuid;not null;primaryKey" json:"artifact_id"`
	Role        string    `gorm:"not null;primaryKey" json:"role"` // e.g., 'primary-image','sbom','docs'
	ArtifactKey *string   `gorm:"uniqueIndex:ux_release_artifact_key,priority:2,where:artifact_key IS NOT NULL" json:"artifact_key,omitempty"` // blueprint artifact id, e.g., 'main'
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName specifies the table name
func (ReleaseArtifact) TableName() string {
	return "release_artifact"
}
package artifact

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

// Artifact represents outputs produced by builds (images, indices, assets, sboms)
type Artifact struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	BuildID   uuid.UUID  `gorm:"type:uuid;not null" json:"build_id"`
	Kind      string     `gorm:"not null" json:"kind"` // e.g., 'oci-image','oci-index','github-asset','s3-object','sbom'
	Name      *string    `json:"name,omitempty"`        // display name/repository/filename
	URI       *string    `json:"uri,omitempty"`
	MediaType *string    `json:"media_type,omitempty"`
	Digest    *string    `gorm:"uniqueIndex:ux_artifact_digest,where:digest IS NOT NULL" json:"digest,omitempty"`
	SizeBytes *int64     `json:"size_bytes,omitempty"`
	Labels    JSONB      `gorm:"type:jsonb" json:"labels,omitempty"`
	Metadata  JSONB      `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

// TableName specifies the table name
func (Artifact) TableName() string {
	return "artifact"
}

// BeforeCreate hook to set UUID if not provided
func (a *Artifact) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
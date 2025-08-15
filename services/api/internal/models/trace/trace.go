package trace

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/enums"
)

// Trace represents a correlation root (PR check, merge build, deploy, etc.)
type Trace struct {
	ID             uuid.UUID             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Purpose        enums.TracePurpose    `gorm:"not null" json:"purpose"`
	RetentionClass enums.RetentionClass  `gorm:"not null;default:'long'" json:"retention_class"`
	RepoID         *uuid.UUID            `gorm:"type:uuid" json:"repo_id,omitempty"`
	Branch         *string               `json:"branch,omitempty"`
	CreatedBy      *string               `json:"created_by,omitempty"`
	CreatedAt      time.Time             `gorm:"not null;default:now()" json:"created_at"`
}

// TableName specifies the table name
func (Trace) TableName() string {
	return "trace"
}

// BeforeCreate hook to set UUID if not provided
func (t *Trace) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
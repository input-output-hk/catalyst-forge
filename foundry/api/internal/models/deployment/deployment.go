package deployment

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/enums"
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

// Deployment represents promotion of a Release to an Environment (GitOps-driven)
type Deployment struct {
	ID             uuid.UUID               `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TraceID        *uuid.UUID              `gorm:"type:uuid" json:"trace_id,omitempty"`
	ReleaseID      uuid.UUID               `gorm:"type:uuid;not null" json:"release_id"`
	EnvID          uuid.UUID               `gorm:"type:uuid;not null;index:ix_deployment_env_proj_created,priority:1" json:"env_id"`
	ProjectID      uuid.UUID               `gorm:"type:uuid;not null;index:ix_deployment_env_proj_created,priority:2" json:"project_id"`
	IntentRevision *string                 `json:"intent_revision,omitempty"` // Git commit SHA written to GitOps repo
	IntentDigest   *string                 `json:"intent_digest,omitempty"`   // hash of canonical intent
	IntentJSON     JSONB                   `gorm:"type:jsonb" json:"intent_json,omitempty"`
	Status         enums.DeploymentStatus  `gorm:"not null" json:"status"`
	LastError      *string                 `json:"last_error,omitempty"`
	CreatedBy      *string                 `json:"created_by,omitempty"`
	CreatedAt      time.Time               `gorm:"not null;default:now();index:ix_deployment_env_proj_created,priority:3,sort:desc" json:"created_at"`
	UpdatedAt      time.Time               `gorm:"not null;default:now()" json:"updated_at"`

	// Relationships
	RenderJob *RenderJob `gorm:"foreignKey:DeploymentID" json:"render_job,omitempty"`
}

// TableName specifies the table name
func (Deployment) TableName() string {
	return "deployment"
}

// BeforeCreate hook to set UUID if not provided
func (d *Deployment) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
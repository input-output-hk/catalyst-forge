package argo

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

// ArgoSync represents Argo CD application status snapshots for observability
type ArgoSync struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeploymentID *uuid.UUID `gorm:"type:uuid" json:"deployment_id,omitempty"`
	EnvID        uuid.UUID  `gorm:"type:uuid;not null;index:ix_argo_sync_env_app_time,priority:1" json:"env_id"`
	AppName      string     `gorm:"not null;index:ix_argo_sync_env_app_time,priority:2" json:"app_name"`
	ObservedRev  *string    `json:"observed_rev,omitempty"`
	SyncStatus   *string    `json:"sync_status,omitempty"`
	HealthStatus *string    `json:"health_status,omitempty"`
	ObservedAt   time.Time  `gorm:"not null;default:now();index:ix_argo_sync_env_app_time,priority:3,sort:desc" json:"observed_at"`
	Raw          JSONB      `gorm:"type:jsonb" json:"raw,omitempty"`
}

// TableName specifies the table name
func (ArgoSync) TableName() string {
	return "argo_sync"
}

// BeforeCreate hook to set UUID if not provided
func (a *ArgoSync) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
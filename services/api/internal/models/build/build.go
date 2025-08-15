package build

import (
	"database/sql/driver"
	"encoding/json"
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

// Build represents a CI build execution for a project at a commit
type Build struct {
	ID            uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TraceID       *uuid.UUID         `gorm:"type:uuid" json:"trace_id,omitempty"`
	RepoID        uuid.UUID          `gorm:"type:uuid;not null" json:"repo_id"`
	ProjectID     uuid.UUID          `gorm:"type:uuid;not null" json:"project_id"`
	CommitSHA     string             `gorm:"not null" json:"commit_sha"`
	Branch        *string            `json:"branch,omitempty"`
	WorkflowRunID *string            `json:"workflow_run_id,omitempty"`
	Status        enums.BuildStatus  `gorm:"not null" json:"status"`
	RunnerEnv     JSONB              `gorm:"type:jsonb" json:"runner_env,omitempty"`
	StartedAt     time.Time          `gorm:"not null;default:now()" json:"started_at"`
	FinishedAt    *time.Time         `json:"finished_at,omitempty"`
	CreatedAt     time.Time          `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time          `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name
func (Build) TableName() string {
	return "build"
}

// BeforeCreate hook to set UUID if not provided
func (b *Build) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
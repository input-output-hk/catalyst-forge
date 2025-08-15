package project

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProjectStatus represents the status of a project
type ProjectStatus string

const (
	ProjectStatusActive  ProjectStatus = "active"
	ProjectStatusRemoved ProjectStatus = "removed"
)

// Project represents a logical project discovered from blueprint within a repo
type Project struct {
	ID                   uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RepoID               uuid.UUID      `gorm:"type:uuid;not null" json:"repo_id"`
	Path                 string         `gorm:"not null" json:"path"`                          // Repo-relative directory for project root
	Slug                 string         `gorm:"not null" json:"slug"`
	DisplayName          *string        `json:"display_name,omitempty"`
	Status               ProjectStatus  `gorm:"not null;default:'active'" json:"status"`
	BlueprintFingerprint *string        `json:"blueprint_fingerprint,omitempty"`
	FirstSeenCommit      *string        `json:"first_seen_commit,omitempty"`
	LastSeenCommit       *string        `json:"last_seen_commit,omitempty"`
	CreatedAt            time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName specifies the table name
func (Project) TableName() string {
	return "project"
}

// BeforeCreate hook to set UUID if not provided
func (p *Project) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
package gitops

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PointerType represents the pointer type for GitOps changes
type PointerType string

const (
	PointerTypeRelease  PointerType = "release"
	PointerTypeRendered PointerType = "rendered"
)

// GitOpsChange represents Git change created for a Deployment (commit/PR)
type GitOpsChange struct {
	ID           uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeploymentID uuid.UUID    `gorm:"type:uuid;not null" json:"deployment_id"`
	Repo         string       `gorm:"not null" json:"repo"`
	Branch       string       `gorm:"not null" json:"branch"`
	CommitSHA    string       `gorm:"not null;index:ix_gitops_change_commit" json:"commit_sha"`
	ChangePath   string       `gorm:"not null" json:"change_path"`
	PRNumber     *int         `json:"pr_number,omitempty"`
	MergedAt     *time.Time   `json:"merged_at,omitempty"`
	PointerType  *PointerType `json:"pointer_type,omitempty"`
	PointerRef   *string      `json:"pointer_ref,omitempty"`
	PointerDigest *string     `json:"pointer_digest,omitempty"`
	CreatedAt    time.Time    `gorm:"not null;default:now()" json:"created_at"`
}

// TableName specifies the table name
func (GitOpsChange) TableName() string {
	return "gitops_change"
}

// BeforeCreate hook to set UUID if not provided
func (g *GitOpsChange) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}
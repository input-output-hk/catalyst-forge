package models

import (
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type GHARepositoryAuth struct {
	ID          uint           `gorm:"primaryKey"          json:"id"`
	Repository  string         `gorm:"not null;uniqueIndex" json:"repository"`
	Permissions pq.StringArray `gorm:"type:text[];not null" json:"permissions"` // ← changed
	Enabled     bool           `gorm:"not null;default:true" json:"enabled"`
	Description string         `json:"description,omitempty"`
	CreatedBy   string         `gorm:"not null" json:"created_by"`
	UpdatedBy   string         `gorm:"not null" json:"updated_by"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (GHARepositoryAuth) TableName() string { return "gha_repository_auths" }

// ----- helpers --------------------------------------------------------------

func (g *GHARepositoryAuth) GetPermissions() []auth.Permission {
	out := make([]auth.Permission, len(g.Permissions))
	for i, p := range g.Permissions {
		out[i] = auth.Permission(p)
	}
	return out
}

func (g *GHARepositoryAuth) SetPermissions(perms []auth.Permission) {
	g.Permissions = make(pq.StringArray, len(perms))
	for i, p := range perms {
		g.Permissions[i] = string(p)
	}
}

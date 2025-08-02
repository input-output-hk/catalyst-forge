package user

import (
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Role represents a role in the system
type Role struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null;uniqueIndex" json:"name"`
	Permissions pq.StringArray `gorm:"type:text[];not null" json:"permissions"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the Role model
func (Role) TableName() string {
	return "roles"
}

// ----- helpers --------------------------------------------------------------

func (r *Role) GetPermissions() []auth.Permission {
	out := make([]auth.Permission, len(r.Permissions))
	for i, p := range r.Permissions {
		out[i] = auth.Permission(p)
	}
	return out
}

func (r *Role) SetPermissions(perms []auth.Permission) {
	r.Permissions = make(pq.StringArray, len(perms))
	for i, p := range perms {
		r.Permissions[i] = string(p)
	}
}

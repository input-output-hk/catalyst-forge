package user

import (
    "time"

    "github.com/lib/pq"
    "gorm.io/gorm"

    "github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
)

// Role represents a role in the system.
type Role struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null;uniqueIndex" json:"name"`
	Permissions pq.StringArray `gorm:"type:text[];not null" json:"permissions"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the Role model.
func (Role) TableName() string {
    return "roles"
}

// ----- helpers --------------------------------------------------------------

// GetPermissions returns the permissions as typed auth.Permission values.
func (r *Role) GetPermissions() []auth.Permission {
    out := make([]auth.Permission, len(r.Permissions))
    for i, p := range r.Permissions {
        out[i] = auth.Permission(p)
    }
    return out
}

// SetPermissions sets the permissions from typed auth.Permission values.
func (r *Role) SetPermissions(perms []auth.Permission) {
    r.Permissions = make(pq.StringArray, len(perms))
    for i, p := range perms {
        r.Permissions[i] = string(p)
    }
}

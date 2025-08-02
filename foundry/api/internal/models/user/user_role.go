package user

import (
	"time"

	"gorm.io/gorm"
)

// UserRole represents a many-to-many relationship between users and roles
type UserRole struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"not null;index" json:"user_id"`
	RoleID uint `gorm:"not null;index" json:"role_id"`

	// Relationships
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role *Role `gorm:"foreignKey:RoleID" json:"role,omitempty"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the UserRole model
func (UserRole) TableName() string {
	return "user_roles"
}

package user

import (
	"time"

	"gorm.io/gorm"
)

// UserStatus type for user status
type UserStatus string

// Possible user statuses
const (
	UserStatusPending  UserStatus = "pending"
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

// User represents a user in the system
type User struct {
	ID     uint       `gorm:"primaryKey" json:"id"`
	Email  string     `gorm:"not null;uniqueIndex" json:"email"`
	Status UserStatus `gorm:"not null;type:string;default:'pending'" json:"status"`

    // New fields for invite-based auth
    EmailVerifiedAt *time.Time `gorm:"index" json:"email_verified_at,omitempty"`
    UserVer         int        `gorm:"not null;default:1" json:"user_ver"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the User model
func (User) TableName() string {
	return "users"
}

package user

import (
    "time"

    "github.com/lib/pq"
    "gorm.io/gorm"
)

// Invite represents an invitation for a new user, including roles and expiry.
type Invite struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Email           string         `gorm:"not null;index" json:"email"`
	Roles           pq.StringArray `gorm:"type:text[];not null" json:"roles"`
	TokenHash       string         `gorm:"not null" json:"-"`
	ExpiresAt       time.Time      `gorm:"not null" json:"expires_at"`
	RedeemedAt      *time.Time     `json:"redeemed_at,omitempty"`
	CreatedBy       uint           `gorm:"not null" json:"created_by"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Rate limiting fields
	FailedAttempts  int        `gorm:"default:0" json:"-"`
	LastAttemptAt   *time.Time `json:"-"`
	LockedUntil     *time.Time `json:"-"`
}

// TableName specifies the table name for the Invite model.
func (Invite) TableName() string { return "invites" }

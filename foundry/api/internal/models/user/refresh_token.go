package user

import (
	"time"

	"gorm.io/gorm"
)

// RefreshToken is an opaque rotating token with reuse detection
type RefreshToken struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"not null;index" json:"user_id"`
	DeviceID   *uint          `gorm:"index" json:"device_id,omitempty"`
	TokenHash  string         `gorm:"not null;uniqueIndex" json:"-"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	LastUsedAt *time.Time     `json:"last_used_at,omitempty"`
	ExpiresAt  time.Time      `gorm:"not null" json:"expires_at"`
	ReplacedBy *uint          `gorm:"index" json:"replaced_by,omitempty"`
	RevokedAt  *time.Time     `json:"revoked_at,omitempty"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

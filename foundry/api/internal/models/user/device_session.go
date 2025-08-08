package user

import (
	"time"

	"gorm.io/gorm"
)

// DeviceSession represents a pending/approved device authorization session
type DeviceSession struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	DeviceCode      string     `gorm:"not null;uniqueIndex" json:"-"`
	UserCode        string     `gorm:"not null;uniqueIndex" json:"user_code"`
	ExpiresAt       time.Time  `gorm:"not null" json:"expires_at"`
	IntervalSeconds int        `gorm:"not null" json:"interval_seconds"`
	Status          string     `gorm:"not null;index" json:"status"` // pending, approved, denied
	ApprovedUserID  *uint      `gorm:"index" json:"approved_user_id,omitempty"`
	LastPolledAt    *time.Time `json:"last_polled_at,omitempty"`
	PollCount       int        `gorm:"not null;default:0" json:"poll_count"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	// optional client metadata from init
	Name        string         `json:"name"`
	Platform    string         `json:"platform"`
	Fingerprint string         `json:"fingerprint"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (DeviceSession) TableName() string { return "device_sessions" }

package user

import (
    "time"

    "gorm.io/gorm"
)

// Device represents a user device that can hold refresh tokens and keys
type Device struct {
    ID          uint           `gorm:"primaryKey" json:"id"`
    UserID      uint           `gorm:"not null;index" json:"user_id"`
    Name        string         `json:"name"`
    Platform    string         `json:"platform"`
    Fingerprint string         `json:"fingerprint"`
    CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
    LastSeenAt  *time.Time     `json:"last_seen_at,omitempty"`
    DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Device) TableName() string { return "devices" }


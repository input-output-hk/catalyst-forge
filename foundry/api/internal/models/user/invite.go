package user

import (
    "time"

    "gorm.io/gorm"
)

type Invite struct {
    ID         uint           `gorm:"primaryKey" json:"id"`
    Email      string         `gorm:"not null;index" json:"email"`
    Role       string         `gorm:"not null" json:"role"`
    TokenHash  string         `gorm:"not null" json:"-"`
    ExpiresAt  time.Time      `gorm:"not null" json:"expires_at"`
    RedeemedAt *time.Time     `json:"redeemed_at,omitempty"`
    CreatedBy  uint           `gorm:"not null" json:"created_by"`
    CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
    DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Invite) TableName() string { return "invites" }


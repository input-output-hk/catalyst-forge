package user

import (
	"time"

	"gorm.io/gorm"
)

// BootstrapToken tracks usage of bootstrap tokens for initial admin setup.
type BootstrapToken struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	TokenHash   string         `gorm:"not null;uniqueIndex" json:"-"` // SHA256 hash of the token
	UsedAt      time.Time      `gorm:"not null" json:"used_at"`
	UsedByEmail string         `gorm:"not null" json:"used_by_email"`
	InviteID    uint           `gorm:"not null" json:"invite_id"` // Reference to the created invite
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the BootstrapToken model.
func (BootstrapToken) TableName() string { return "bootstrap_tokens" }

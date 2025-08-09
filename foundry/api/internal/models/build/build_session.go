package build

import (
	"time"
)

// BuildSession represents a CI/build session for provenance and rate limiting
type BuildSession struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	OwnerType string    `gorm:"size:32" json:"owner_type"`
	OwnerID   uint      `gorm:"index" json:"owner_id"`
	Source    string    `gorm:"size:64" json:"source"`
	Metadata  []byte    `gorm:"type:jsonb" json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

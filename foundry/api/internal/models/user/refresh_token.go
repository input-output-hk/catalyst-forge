package user

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a family-based rotating refresh token with replay detection.
type RefreshToken struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	DeviceID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"device_id"`
	FamilyID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"family_id"`
	ParentID   *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	SecretHash string     `gorm:"not null;size:64" json:"-"` // HMAC(REFRESH_HASH_SECRET, secret)
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"created_at"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	RotatedAt  *time.Time `gorm:"index" json:"rotated_at,omitempty"` // When this token was rotated/used

	// Client metadata for security analysis
	IP        *string `gorm:"type:inet" json:"-"`
	UserAgent *string `gorm:"size:255" json:"-"`

	// Relationships
	User   User   `gorm:"foreignKey:UserID" json:"-"`
	Device Device `gorm:"foreignKey:DeviceID" json:"-"`
}

// RefreshToken status constants.
const (
    RefreshTokenStatusActive  = "active"
    RefreshTokenStatusRevoked = "revoked"
)

// TableName specifies the table name for the RefreshToken model.
func (RefreshToken) TableName() string { return "auth_refresh_tokens" }

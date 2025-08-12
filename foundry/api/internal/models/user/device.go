package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Device represents a browser device registered for authentication with ECDSA keypairs.
type Device struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID        uint           `gorm:"not null;index" json:"user_id"`
	Name          string         `gorm:"not null" json:"name"`
	PublicJWK     datatypes.JSON `gorm:"type:jsonb;not null" json:"-"`
	JWKThumbprint string         `gorm:"uniqueIndex;size:128;not null" json:"-"`
	Status        string         `gorm:"index;default:'active';not null" json:"status"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	LastUsedAt    *time.Time     `json:"last_used_at,omitempty"`
	RevokedAt     *time.Time     `json:"revoked_at,omitempty"`

	// Relationships
	User          User           `gorm:"foreignKey:UserID" json:"-"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:DeviceID" json:"-"`
}

// DeviceStatus constants.
const (
    DeviceStatusActive   = "active"
    DeviceStatusRevoked  = "revoked"
    DeviceStatusDisabled = "disabled"
)

// TableName specifies the table name for the Device model.
func (Device) TableName() string { return "auth_devices" }

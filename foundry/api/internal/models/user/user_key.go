package user

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
	"gorm.io/gorm"
)

// UserKeyStatus type for user key status
type UserKeyStatus string

// Possible user key statuses
const (
	UserKeyStatusActive   UserKeyStatus = "active"
	UserKeyStatusInactive UserKeyStatus = "inactive"
	UserKeyStatusRevoked  UserKeyStatus = "revoked"
)

// UserKey represents an Ed25519 key belonging to a user
type UserKey struct {
	ID        uint          `gorm:"primaryKey" json:"id"`
	UserID    uint          `gorm:"not null;index" json:"user_id"`
	Kid       string        `gorm:"not null;uniqueIndex" json:"kid"`
	PubKeyB64 string        `gorm:"not null" json:"pubkey_b64"`
	Status    UserKeyStatus `gorm:"not null;type:string;default:'active'" json:"status"`

	// Relationships
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the UserKey model
func (UserKey) TableName() string {
	return "user_keys"
}

// ToKeyPair converts the UserKey to a KeyPair using the PubKeyB64 field.
func (uk *UserKey) ToKeyPair() (*auth.KeyPair, error) {
	// Decode the base64 public key
	pubKeyBytes, err := base64.StdEncoding.DecodeString(uk.PubKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	// Convert to ed25519.PublicKey
	pubKey := ed25519.PublicKey(pubKeyBytes)

	return &auth.KeyPair{
		PublicKey: pubKey,
	}, nil
}

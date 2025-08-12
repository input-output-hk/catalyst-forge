package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	// RefreshCookieName is the name of the refresh token cookie.
	RefreshCookieName = "__Host-refresh-token"
)

// User represents an authenticated user in the system.
type User struct {
	ID             uuid.UUID
	Email          string
	Roles          []string
	Permissions    []string   // Derived permissions from roles
	SessionVersion int64
	SuspendedAt    *time.Time // Account suspension timestamp
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Credential represents a WebAuthn credential registered to a user.
type Credential struct {
	ID          []byte    // credentialId from WebAuthn
	UserID      uuid.UUID
	PublicKey   []byte    // COSE public key
	AAGUID      string    // Authenticator AAGUID
	DeviceName  string    // User-friendly device name
	RK          bool      // Resident Key capable
	Transports  []string  // USB, NFC, BLE, internal
	SignCount   uint32    // Counter for clone detection
	CreatedAt   time.Time
	LastUsedAt  time.Time
	Revoked     bool
}

// Invite represents an invitation for a new user to onboard.
type Invite struct {
	ID         uuid.UUID
	Email      string
	Roles      []string
	TokenHash  []byte    // HMAC-SHA256 of the invite token
	ExpiresAt  time.Time
	Attempts   int       // Failed redemption attempts
	RedeemedAt *time.Time
	CreatedBy  uuid.UUID // Admin who created the invite
}

// RecoveryCode represents a single-use recovery code for account recovery.
type RecoveryCode struct {
	UserID    uuid.UUID
	Hash      []byte    // SHA256 of the recovery code
	UsedAt    *time.Time
	CreatedAt time.Time
}

// RefreshToken represents a refresh token with rotation tracking.
type RefreshToken struct {
	ID             uuid.UUID
	FamilyID       uuid.UUID // Groups related tokens for rotation
	UserID         uuid.UUID
	Hash           []byte    // SHA256 of the token
	SessionVersion int64     // User's session version at creation
	CreatedAt      time.Time
	ExpiresAt      time.Time  // When this token expires
	RotatedAt      *time.Time // When this token was rotated to a new one
	RevokedAt      *time.Time // When this token was explicitly revoked
	Reason         *string    // Reason for revocation
}
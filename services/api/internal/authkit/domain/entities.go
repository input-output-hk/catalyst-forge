package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	// RefreshCookieName is the name of the refresh token cookie.
	// Standardize on underscore per httpkit helpers and cookie naming guidance.
	RefreshCookieName = "__Host-refresh_token"
)

// User represents an authenticated user in the system.
type User struct {
	ID             uuid.UUID
	Email          string
	Roles          []string
	Permissions    []string // Derived permissions from roles
	SessionVersion int64
	SuspendedAt    *time.Time // Account suspension timestamp
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Credential represents a WebAuthn credential registered to a user.
type Credential struct {
	ID         []byte // credentialId from WebAuthn
	UserID     uuid.UUID
	PublicKey  []byte   // COSE public key
	AAGUID     string   // Authenticator AAGUID
	DeviceName string   // User-friendly device name
	RK         bool     // Resident Key capable
	Transports []string // USB, NFC, BLE, internal
	SignCount  uint32   // Counter for clone detection
	CreatedAt  time.Time
	LastUsedAt time.Time
	Revoked    bool
}

// Invite represents an invitation for a new user to onboard.
type Invite struct {
	ID         uuid.UUID
	Email      string
	Roles      []string
	TokenHash  []byte // HMAC-SHA256 of the invite token
	ExpiresAt  time.Time
	Attempts   int // Failed redemption attempts
	RedeemedAt *time.Time
	CreatedBy  uuid.UUID // Admin who created the invite
}

// RecoveryCode represents a single-use recovery code for account recovery.
type RecoveryCode struct {
	UserID    uuid.UUID
	Hash      []byte // SHA256 of the recovery code
	UsedAt    *time.Time
	CreatedAt time.Time
}

// RefreshToken represents a refresh token with rotation tracking.
type RefreshToken struct {
	ID             uuid.UUID
	FamilyID       uuid.UUID  // Groups related tokens for rotation
	UserID         uuid.UUID
	DeviceID       *uuid.UUID // Optional: CLI device that owns this token
	Hash           []byte     // SHA256 of the token
	SessionVersion int64      // User's session version at creation
	CreatedAt      time.Time
	ExpiresAt      time.Time  // When this token expires
	RotatedAt      *time.Time // When this token was rotated to a new one
	RevokedAt      *time.Time // When this token was explicitly revoked
	Reason         *string    // Reason for revocation
}

// Device represents a CLI device that has been linked to a user account.
type Device struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DeviceName string    // User-friendly name like "Paul's MBP • Forge CLI"
	CreatedAt  time.Time
	LastUsedAt time.Time
	RevokedAt  *time.Time // When device was revoked/deleted
}

// DeviceLink represents a pending device authorization flow.
type DeviceLink struct {
	ID           uuid.UUID
	DeviceCode   []byte    // Hashed device code (long, opaque)
	UserCode     string    // Short, human-friendly code like "J7FQ-K9"
	DeviceName   string    // Device name provided by CLI
	Purpose      string    // "login" or "step_up"
	UserID       *uuid.UUID // Set after authorization
	AuthorizedAt *time.Time // When user authorized in browser
	ExpiresAt    time.Time  // TTL for the flow (10 min default)
	CreatedAt    time.Time
	Interval     int       // Polling interval in seconds (5s default)
}

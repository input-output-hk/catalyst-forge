package gormstore

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the database.
type User struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Email          string     `gorm:"uniqueIndex;not null"`
	FullName       string     `gorm:"type:text"`
	RolesJSON      string     `gorm:"type:text;column:roles"` // JSON array of roles
	SessionVersion int64      `gorm:"not null;default:1"`
	SuspendedAt    *time.Time `gorm:"index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name for User.
func (User) TableName() string {
	return "auth_users"
}

// Credential represents a WebAuthn credential in the database.
type Credential struct {
	ID         []byte    `gorm:"primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;index;not null"`
	PublicKey  []byte    `gorm:"not null"`
	AAGUID     string    `gorm:"size:36"`
	DeviceName string    `gorm:"not null"`
	RK         bool      `gorm:"not null;default:false"`
	Transports string    `gorm:"type:text"` // JSON array or CSV
	SignCount  uint32    `gorm:"not null;default:0"`
	CreatedAt  time.Time
	LastUsedAt time.Time
	Revoked    bool `gorm:"not null;default:false;index"`

	// Foreign key relation
	User User `gorm:"foreignKey:UserID;references:ID"`
}

// TableName specifies the table name for Credential.
func (Credential) TableName() string {
	return "auth_credentials"
}

// Invite represents an invitation in the database.
type Invite struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Email      string     `gorm:"index;not null"`
	RolesJSON  string     `gorm:"type:text;column:roles"` // JSON array of roles
	TokenHash  []byte     `gorm:"not null;uniqueIndex"`
	ExpiresAt  time.Time  `gorm:"index;not null"`
	Attempts   int        `gorm:"not null;default:0"`
	RedeemedAt *time.Time `gorm:"index"`
	CreatedBy  uuid.UUID  `gorm:"type:uuid;not null"`
	CreatedAt  time.Time

	// Foreign key relation
	Creator User `gorm:"foreignKey:CreatedBy;references:ID"`
}

// TableName specifies the table name for Invite.
func (Invite) TableName() string {
	return "auth_invites"
}

// AccessRequest stores access requests submitted from the landing page.
type AccessRequest struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Email     string     `gorm:"uniqueIndex;not null"`
	Reason    string     `gorm:"type:text"`
	Status    string     `gorm:"index;not null;default:pending"`
	Attempts  int        `gorm:"not null;default:1"`
	DecidedAt *time.Time `gorm:"index"`
	DecidedBy *uuid.UUID `gorm:"type:uuid;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (AccessRequest) TableName() string { return "auth_access_requests" }

// RecoveryCode represents a recovery code in the database.
type RecoveryCode struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;index;not null"`
	Hash      []byte     `gorm:"not null;uniqueIndex"`
	UsedAt    *time.Time `gorm:"index"`
	CreatedAt time.Time

	// Foreign key relation
	User User `gorm:"foreignKey:UserID;references:ID"`
}

// TableName specifies the table name for RecoveryCode.
func (RecoveryCode) TableName() string {
	return "auth_recovery_codes"
}

// RefreshToken represents a refresh token in the database.
type RefreshToken struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	FamilyID       uuid.UUID  `gorm:"type:uuid;index;not null"`
	UserID         uuid.UUID  `gorm:"type:uuid;index;not null"`
	DeviceID       *uuid.UUID `gorm:"type:uuid;index"` // Optional CLI device
	Hash           []byte     `gorm:"not null;uniqueIndex"`
	SessionVersion int64      `gorm:"not null"`
	CreatedAt      time.Time
	ExpiresAt      time.Time  `gorm:"index;not null"`
	RotatedAt      *time.Time `gorm:"index"`
	RevokedAt      *time.Time `gorm:"index"`
	Reason         *string
	UserAgent      string
	IPAddress      string

	// Foreign key relations
	User   User    `gorm:"foreignKey:UserID;references:ID"`
	Device *Device `gorm:"foreignKey:DeviceID;references:ID"`
}

// TableName specifies the table name for RefreshToken.
func (RefreshToken) TableName() string {
	return "auth_refresh_tokens"
}

// Device represents a CLI device linked to a user account.
type Device struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;index;not null"`
	DeviceName string    `gorm:"not null"`
	CreatedAt  time.Time
	LastUsedAt time.Time
	RevokedAt  *time.Time `gorm:"index"`

	// Foreign key relation
	User User `gorm:"foreignKey:UserID;references:ID"`
}

// TableName specifies the table name for Device.
func (Device) TableName() string {
	return "auth_devices"
}

// DeviceLink represents a pending device authorization flow.
type DeviceLink struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	DeviceCode   []byte     `gorm:"not null;uniqueIndex"`         // Hashed device code
	UserCode     string     `gorm:"size:10;uniqueIndex;not null"` // Short user code
	DeviceName   string     `gorm:"not null"`
	Purpose      string     `gorm:"size:20;not null"` // "login" or "step_up"
	UserID       *uuid.UUID `gorm:"type:uuid;index"`
	AuthorizedAt *time.Time
	ExpiresAt    time.Time `gorm:"index;not null"`
	CreatedAt    time.Time
	Interval     int `gorm:"not null;default:5"` // Polling interval in seconds

	// Foreign key relation (optional, set after authorization)
	User *User `gorm:"foreignKey:UserID;references:ID"`
}

// TableName specifies the table name for DeviceLink.
func (DeviceLink) TableName() string {
	return "auth_device_links"
}

// BootstrapTokenUsed records consumption of the one-time bootstrap token.
type BootstrapTokenUsed struct {
	TokenHashHex string    `gorm:"primaryKey;size:64;uniqueIndex"`
	UsedByEmail  string    `gorm:"not null"`
	UsedAt       time.Time `gorm:"not null;index"`
}

// TableName specifies the table name for BootstrapTokenUsed.
func (BootstrapTokenUsed) TableName() string { return "auth_bootstrap_tokens" }

// AuditEvent represents an audit event in the database.
type AuditEvent struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Type      string     `gorm:"index;not null"`
	UserID    *uuid.UUID `gorm:"type:uuid;index"`
	ActorID   *uuid.UUID `gorm:"type:uuid;index"`
	IPAddress string
	UserAgent string
	Metadata  JSON      `gorm:"type:jsonb"`
	CreatedAt time.Time `gorm:"index"`
}

// TableName specifies the table name for AuditEvent.
func (AuditEvent) TableName() string {
	return "auth_audit_events"
}

// JSON is a custom type for JSONB fields.
type JSON map[string]interface{}

// Value implements driver.Valuer interface.
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface.
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		bytes = []byte(value.(string))
	}
	return json.Unmarshal(bytes, j)
}

// StringSlice is a custom type for storing string slices as JSON.
type StringSlice []string

// Value implements driver.Valuer interface.
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan implements sql.Scanner interface.
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		bytes = []byte(value.(string))
	}
	return json.Unmarshal(bytes, s)
}

// AutoMigrate runs GORM auto migration for all auth models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Credential{},
		&Invite{},
		&AccessRequest{},
		&RecoveryCode{},
		&Device{},       // Added for CLI device management
		&DeviceLink{},   // Added for device-code linking flow
		&RefreshToken{}, // Updated with DeviceID foreign key
		&AuditEvent{},
		&BootstrapTokenUsed{},
		&GithubPolicy{},
	)
}

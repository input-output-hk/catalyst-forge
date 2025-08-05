package users

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Roles []Role    `json:"roles,omitempty"`
	Keys  []UserKey `json:"keys,omitempty"`
}

// Role represents a role in the system
type Role struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	Users []User `json:"users,omitempty"`
}

// UserRole represents a user-role relationship
type UserRole struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	RoleID    uint      `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	User *User `json:"user,omitempty"`
	Role *Role `json:"role,omitempty"`
}

// UserKey represents a user's authentication key
type UserKey struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	Kid       string    `json:"kid"` // Key ID
	PubKeyB64 string    `json:"pubkey_b64"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	User *User `json:"user,omitempty"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Email  string `json:"email"`
	Status string `json:"status,omitempty"`
}

// RegisterUserRequest represents the request body for registering a user
type RegisterUserRequest struct {
	Email string `json:"email"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Email  string `json:"email"`
	Status string `json:"status,omitempty"`
}

// CreateRoleRequest represents the request body for creating a role
type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleRequest represents the request body for updating a role
type UpdateRoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// CreateUserKeyRequest represents the request body for creating a user key
type CreateUserKeyRequest struct {
	UserID    uint   `json:"user_id"`
	Kid       string `json:"kid"`
	PubKeyB64 string `json:"pubkey_b64"`
	Status    string `json:"status,omitempty"`
}

// RegisterUserKeyRequest represents the request body for registering a user key
type RegisterUserKeyRequest struct {
	Email     string `json:"email"`
	Kid       string `json:"kid"`
	PubKeyB64 string `json:"pubkey_b64"`
}

// UpdateUserKeyRequest represents the request body for updating a user key
type UpdateUserKeyRequest struct {
	UserID    *uint   `json:"user_id,omitempty"`
	Kid       *string `json:"kid,omitempty"`
	PubKeyB64 *string `json:"pubkey_b64,omitempty"`
	Status    *string `json:"status,omitempty"`
}

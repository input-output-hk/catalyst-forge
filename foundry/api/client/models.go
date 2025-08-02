package client

import (
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
)

// DeploymentStatus type for deployment status
type DeploymentStatus string

// Possible deployment statuses
const (
	DeploymentStatusPending   DeploymentStatus = "pending"
	DeploymentStatusRunning   DeploymentStatus = "running"
	DeploymentStatusSucceeded DeploymentStatus = "succeeded"
	DeploymentStatusFailed    DeploymentStatus = "failed"
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

// ChallengeRequest represents the request body for creating a challenge
type ChallengeRequest struct {
	Email string `json:"email"`
	Kid   string `json:"kid"`
}

// LoginResponse represents the response body for authentication
type LoginResponse struct {
	Token string `json:"token"`
}

// UpdateUserKeyRequest represents the request body for updating a user key
type UpdateUserKeyRequest struct {
	UserID    *uint   `json:"user_id,omitempty"`
	Kid       *string `json:"kid,omitempty"`
	PubKeyB64 *string `json:"pubkey_b64,omitempty"`
	Status    *string `json:"status,omitempty"`
}

// GHARepositoryAuth represents the authentication configuration for a GitHub repository
type GHARepositoryAuth struct {
	ID          uint      `json:"id"`
	Repository  string    `json:"repository"`  // Format: "owner/repo"
	Permissions []string  `json:"permissions"` // Array of permission strings
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description,omitempty"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ValidateTokenRequest represents the request body for token validation
type ValidateTokenRequest struct {
	Token    string `json:"token"`
	Audience string `json:"audience,omitempty"`
}

// ValidateTokenResponse represents the response body for token validation
type ValidateTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	UserID    string    `json:"user_id"`
}

// CreateAuthRequest represents the request body for creating a GHA authentication configuration
type CreateAuthRequest struct {
	Repository  string            `json:"repository"`
	Permissions []auth.Permission `json:"permissions"`
	Enabled     bool              `json:"enabled"`
	Description string            `json:"description,omitempty"`
}

// UpdateAuthRequest represents the request body for updating a GHA authentication configuration
type UpdateAuthRequest struct {
	Repository  string            `json:"repository"`
	Permissions []auth.Permission `json:"permissions"`
	Enabled     bool              `json:"enabled"`
	Description string            `json:"description,omitempty"`
}

// Release represents a point-in-time project release
type Release struct {
	ID           string    `json:"id"`
	SourceRepo   string    `json:"source_repo"`
	SourceCommit string    `json:"source_commit"`
	SourceBranch string    `json:"source_branch,omitempty"`
	Project      string    `json:"project"`
	ProjectPath  string    `json:"project_path"`
	Created      time.Time `json:"created"`
	Bundle       string    `json:"bundle"` // Base64-encoded source code

	// Relationships
	Deployments []ReleaseDeployment `json:"deployments,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeploymentEvent represents an event that occurred during a deployment
type DeploymentEvent struct {
	ID           uint      `json:"id"`
	DeploymentID string    `json:"deployment_id"`
	Name         string    `json:"name"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ReleaseDeployment represents a point-in-time deployment of a specific release
type ReleaseDeployment struct {
	ID        string           `json:"id"` // Generated from ReleaseID + timestamp
	ReleaseID string           `json:"release_id"`
	Timestamp time.Time        `json:"timestamp"`
	Status    DeploymentStatus `json:"status"`
	Reason    string           `json:"reason,omitempty"`
	Attempts  int              `json:"attempts"`

	// Relationships
	Release *Release          `json:"release,omitempty"`
	Events  []DeploymentEvent `json:"events,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReleaseAlias represents an alias for a release
type ReleaseAlias struct {
	Name      string    `json:"name"`
	ReleaseID string    `json:"release_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Release *Release `json:"release,omitempty"`
}

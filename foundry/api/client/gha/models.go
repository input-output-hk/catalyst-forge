package gha

import (
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
)

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

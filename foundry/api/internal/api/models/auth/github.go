package auth

import "github.com/google/uuid"

// GithubPolicyCreateRequest defines the payload to create a GitHub policy.
type GithubPolicyCreateRequest struct {
	Repository   string   `json:"repository" binding:"required"`
	Refs         []string `json:"refs,omitempty"`
	Environments []string `json:"environments,omitempty"`
	Workflows    []string `json:"workflows,omitempty"`
	Roles        []string `json:"roles" binding:"required"`
	Enabled      bool     `json:"enabled"`
}

// GithubPolicyUpdateRequest defines the payload to update a GitHub policy.
type GithubPolicyUpdateRequest struct {
	Refs         []string `json:"refs,omitempty"`
	Environments []string `json:"environments,omitempty"`
	Workflows    []string `json:"workflows,omitempty"`
	Roles        []string `json:"roles" binding:"required"`
	Enabled      bool     `json:"enabled"`
}

// GithubPolicyResponse represents a policy record.
type GithubPolicyResponse struct {
	ID           uuid.UUID `json:"id"`
	Repository   string    `json:"repository"`
	Refs         []string  `json:"refs,omitempty"`
	Environments []string  `json:"environments,omitempty"`
	Workflows    []string  `json:"workflows,omitempty"`
	Roles        []string  `json:"roles"`
	Enabled      bool      `json:"enabled"`
}

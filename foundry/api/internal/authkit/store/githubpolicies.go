package store

import (
	"context"

	"github.com/google/uuid"
)

// GithubPolicy describes authorization mapping for a GitHub repository and optional constraints.
type GithubPolicy struct {
	ID           uuid.UUID
	Repository   string
	Refs         []string
	Environments []string
	Workflows    []string
	Roles        []string
	Enabled      bool
}

// GithubPolicyStore provides CRUD operations for GitHub policies.
type GithubPolicyStore interface {
	Create(ctx context.Context, p *GithubPolicy) error
	Update(ctx context.Context, p *GithubPolicy) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*GithubPolicy, error)
	List(ctx context.Context) ([]*GithubPolicy, error)
	// LookupByRepository returns policies for the given repository.
	LookupByRepository(ctx context.Context, repository string) ([]*GithubPolicy, error)
}

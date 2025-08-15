package store

import (
	"context"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/google/uuid"
)

// UserStore handles user persistence.
type UserStore interface {
	// Create creates a new user with the given email and roles.
	Create(ctx context.Context, email string, roles []string) (*domain.User, error)
	
	// GetByEmail retrieves a user by their email address.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	
	// GetByID retrieves a user by their ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	
	// UpdateRoles updates the roles for a user.
	UpdateRoles(ctx context.Context, id uuid.UUID, roles []string) error
	
	// BumpSessionVersion increments the session version to invalidate all sessions.
	BumpSessionVersion(ctx context.Context, id uuid.UUID) error
}
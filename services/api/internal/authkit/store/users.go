package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
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

	// UpdateFullName updates the user's display name.
	UpdateFullName(ctx context.Context, id uuid.UUID, fullName string) error

	// BumpSessionVersion increments the session version to invalidate all sessions.
	BumpSessionVersion(ctx context.Context, id uuid.UUID) error

	// List returns a subset of users ordered by creation time descending.
	// This method is intended for admin listing; filtering and pagination can be
	// applied via limit/offset at the store level and refined at higher layers.
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)

	// ListFiltered returns users filtered by a simple query and role.
	// q filters by email substring (case-insensitive).
	// role filters by presence of a role (e.g., "admin" or "member").
	ListFiltered(ctx context.Context, q string, role string, limit, offset int) ([]*domain.User, error)

	// CountFiltered returns the total number of users matching the filters.
	CountFiltered(ctx context.Context, q string, role string) (int64, error)

	// UpdateSuspended sets or clears the suspended_at timestamp.
	UpdateSuspended(ctx context.Context, id uuid.UUID, suspended bool, at time.Time) error

	// Delete permanently removes a user and related auth data as implemented by the store.
	Delete(ctx context.Context, id uuid.UUID) error
}

// AccessRequestStore handles persistence for access requests.
type AccessRequestStore interface {
	CreateOrBump(ctx context.Context, email string, reason string, now time.Time) (*domain.AccessRequest, error)
	List(ctx context.Context, status string, q string, limit, offset int) ([]*domain.AccessRequest, int64, error)
	Decide(ctx context.Context, id uuid.UUID, approve bool, decidedBy uuid.UUID, note string, now time.Time) error
}

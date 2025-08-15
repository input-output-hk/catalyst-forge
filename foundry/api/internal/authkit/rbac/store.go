package rbac

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Store abstracts RBAC persistence.
type Store interface {
	// Roles
	CreateRole(ctx context.Context, role RoleDef) error
	GetRole(ctx context.Context, slug string) (*RoleDef, error)
	UpdateRole(ctx context.Context, role RoleDef) error
	ListRoles(ctx context.Context) ([]RoleDef, error)
	BumpRoleVersion(ctx context.Context, slug string) error

	// Bindings
	AddBinding(ctx context.Context, b Binding) error
	RemoveBinding(ctx context.Context, id uuid.UUID) error
	ListBindings(ctx context.Context, subj Subject) ([]Binding, error)
	ListBindingsByScope(ctx context.Context, scope ScopeType, scopeID string) ([]Binding, error)

	// Versions for invalidation
	GetPrincipalVersion(ctx context.Context, subj Subject) (int64, error)
	BumpPrincipalVersion(ctx context.Context, subj Subject) error
}

// Cache abstracts RBAC caching.
type Cache interface {
	// Role entries cache: key by slug+version;
	GetRoleEntries(slug string, version int64) ([]RoleEntry, bool)
	SetRoleEntries(slug string, version int64, entries []RoleEntry, ttl time.Duration)

	// Principal cache: key by subject+version+org
	GetPrincipal(key string) ([]RoleEntry, bool)
	SetPrincipal(key string, entries []RoleEntry, ttl time.Duration)
}

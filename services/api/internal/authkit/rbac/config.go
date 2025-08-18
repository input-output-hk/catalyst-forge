package rbac

import "time"

// Config holds module-level RBAC configuration.
type Config struct {
	// EnableDeny controls whether explicit deny entries override allows.
	// Default: true.
	EnableDeny bool

	// Cache TTLs (zero to rely on versioning only).
	RoleCacheTTL      time.Duration
	PrincipalCacheTTL time.Duration

	// Scopes planner determines evaluation order and scope ID extraction.
	// If nil, a default planner that preserves legacy behavior is used.
	Scopes ScopePlanner
}

// DefaultConfig returns sane defaults for RBAC configuration.
func DefaultConfig() Config {
	return Config{
		EnableDeny:        true,
		RoleCacheTTL:      5 * time.Minute,
		PrincipalCacheTTL: 2 * time.Minute,
	}
}

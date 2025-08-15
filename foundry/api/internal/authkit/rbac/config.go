package rbac

import "time"

// Config holds module-level RBAC configuration.
type Config struct {
	// SuperRoles bypass checks entirely when present on a subject.
	SuperRoles []string

	// EnableDeny controls whether explicit deny entries override allows.
	// Default: true.
	EnableDeny bool

	// Cache TTLs (zero to rely on versioning only).
	RoleCacheTTL      time.Duration
	PrincipalCacheTTL time.Duration
}

// DefaultConfig returns sane defaults for RBAC configuration.
func DefaultConfig() Config {
	return Config{
		SuperRoles:        []string{"admin"},
		EnableDeny:        true,
		RoleCacheTTL:      5 * time.Minute,
		PrincipalCacheTTL: 2 * time.Minute,
	}
}

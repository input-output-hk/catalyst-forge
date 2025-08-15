package certkit

import (
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rate"
	rbac "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"
)

// Clock provides time operations (mockable for testing).
type Clock interface {
	Now() time.Time
}

// Logger is a minimal logging interface; prefer stdlib slog in callers.
type Logger interface {
	Error(msg string, kv ...any)
	Info(msg string, kv ...any)
}

// Deps holds external dependencies for certkit.
type Deps struct {
	PCA     PCAClient    // AWS PCA client wrapper
	Limiter rate.Limiter // optional; reuse authkit rate limiter shape
	Clock   Clock
	Logger  Logger
	RBAC    rbac.Manager // optional; for programmatic SAN checks
}

package authkit

import (
	"context"
	"time"

	basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rate"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

// Deps holds all dependencies for the authentication system.
type Deps struct {
	Stores  Stores            // All storage implementations
	Keys    crypto.KeyManager // JWT signing/verification
	Rand    crypto.Rand       // Secure random generation
	Clock   Clock             // Time provider (for testing)
	Limiter rate.Limiter      // Rate limiter (optional, can be no-op)
	CSRF    basehttpkit.CSRF  // CSRF protection provider
	Logger  Logger            // Logging interface
	Mailer  Mailer            // Email service (optional, for recovery)
	KV      store.KV          // Ephemeral storage for step-up grants
}

// Stores groups all storage interfaces.
type Stores struct {
	Users          store.UserStore
	Credentials    store.CredentialStore
	Invites        store.InviteStore
	Access         store.AccessRequestStore
	RecoveryCodes  store.RecoveryCodeStore
	Refresh        store.RefreshStore
	Audit          store.AuditStore
	Challenges     store.ChallengeStore    // For WebAuthn challenges
	Bootstrap      store.BootstrapStore    // For one-time admin bootstrap replay tracking
	GithubPolicies store.GithubPolicyStore // For GitHub OIDC policy management
	DeviceLinks    store.DeviceLinkStore   // For device-code linking flows
	Devices        store.DeviceStore       // For CLI device management
}

// Clock provides time operations (mockable for testing).
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}

// Logger defines the logging interface.
type Logger interface {
	// Deprecated: prefer stdlib slog.Logger. This interface remains only for
	// compatibility during migration and will be removed in a future release.
	// Debug logs a debug message with optional fields.
	Debug(ctx context.Context, msg string, fields ...any)
	// Info logs an info message with optional fields.
	Info(ctx context.Context, msg string, fields ...any)
	// Warn logs a warning message with optional fields.
	Warn(ctx context.Context, msg string, fields ...any)
	// Error logs an error message with optional fields.
	Error(ctx context.Context, msg string, fields ...any)
}

// Mailer defines the email sending interface.
type Mailer interface {
	// SendRecoveryEmail sends an account recovery email with a recovery link.
	SendRecoveryEmail(ctx context.Context, email string, recoveryLink string) error
	// SendInviteEmail sends an invitation email with an invite link.
	SendInviteEmail(ctx context.Context, email string, inviteLink string) error
	// SendSecurityAlert sends a security alert email with details about the event.
	SendSecurityAlert(ctx context.Context, email string, alertType string, details map[string]string) error
}

// defaultClock implements Clock using real time.
type defaultClock struct{}

// Now returns the current time.
func (c defaultClock) Now() time.Time {
	return time.Now()
}

// DefaultClock returns a Clock that uses real time.
func DefaultClock() Clock {
	return defaultClock{}
}

// newTokenServiceForMiddleware constructs a minimal TokenService for authn middleware.
func newTokenServiceForMiddleware(cfg Config, deps Deps) service.TokenService {
	// Default to 30 minutes; Sign is unused by middleware, ParseAccess uses deps.Keys
	return service.NewTokenService(deps.Keys, deps.Rand, cfg.Origin, 30*time.Minute)
}

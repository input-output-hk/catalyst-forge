package authkit

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"os"
	"sync"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/db"
	"github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rate"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store/gormstore"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store/memstore"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/config"
	"gorm.io/gorm"
)

// Deps holds all dependencies for the authentication system.
type Deps struct {
	Stores  Stores            // All storage implementations
	Keys    crypto.KeyManager // JWT signing/verification
	Rand    crypto.Rand       // Secure random generation
	Clock   Clock             // Time provider (for testing)
	Limiter rate.Limiter      // Rate limiter (optional, can be no-op)
	CSRF    httpkit.CSRF      // CSRF protection provider
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

// BuildDeps assembles AuthKit dependencies from API config and infrastructure.
func BuildDeps(ctx context.Context, c *config.Config, dbStore db.Store, gdb *gorm.DB, keys crypto.KeyManager, logger Logger) Deps {
	cookieCfg := httpkit.DefaultCookieConfig()

	// Build CSRF with optional persistent secret
	var csrf httpkit.CSRF = nil
	if c != nil && c.Auth.CSRFSecret != "" {
		// try base64 (std and raw-url), then hex, else raw bytes
		var secret []byte
		if b, err := base64.StdEncoding.DecodeString(c.Auth.CSRFSecret); err == nil {
			secret = b
		} else if b, err := base64.RawURLEncoding.DecodeString(c.Auth.CSRFSecret); err == nil {
			secret = b
		} else if b, err := hex.DecodeString(c.Auth.CSRFSecret); err == nil {
			secret = b
		} else {
			secret = []byte(c.Auth.CSRFSecret)
		}
		csrf = httpkit.NewDoubleSubmitCSRFWithSecret(cookieCfg, 24*time.Hour, secret)
	} else {
		csrf = httpkit.NewDoubleSubmitCSRF(cookieCfg, 24*time.Hour)
	}

	stores := Stores{
		Users:          gormstore.NewUserStore(gdb),
		Credentials:    gormstore.NewCredentialStore(gdb),
		Invites:        gormstore.NewInviteStore(gdb),
		Access:         gormstore.NewAccessRequestStore(gdb),
		RecoveryCodes:  gormstore.NewRecoveryCodeStore(gdb),
		Refresh:        gormstore.NewRefreshStore(gdb),
		Audit:          gormstore.NewAuditStore(gdb),
		Challenges:     memstore.NewChallengeStore(),
		Bootstrap:      gormstore.NewBootstrapTokenStore(gdb),
		GithubPolicies: gormstore.NewGithubPolicyStore(gdb),
		DeviceLinks:    gormstore.NewDeviceLinkStore(gdb),
		Devices:        gormstore.NewDeviceStore(gdb),
	}

	// Keys: prefer provided key manager; if nil, create or load from config
	if keys == nil {
		// Start with a strict manager (prevents KID reuse) and add configured key if present
		km := crypto.NewStrictES256KeyManager()
		if c != nil {
			kid := c.Auth.SigningKeyKID
			if kid == "" {
				kid = "default"
			}
			if s, ok := km.(*crypto.StrictES256KeyManager); ok {
				if c.Auth.SigningKeyPath != "" {
					if pem, err := os.ReadFile(c.Auth.SigningKeyPath); err == nil {
						_ = s.AddKey(kid, pem)
						_ = s.SetCurrentKID(kid)
					}
				} else if c.Auth.SigningKeyPEM != "" {
					_ = s.AddKey(kid, []byte(c.Auth.SigningKeyPEM))
					_ = s.SetCurrentKID(kid)
				}
			}
		}
		keys = km
	}

	deps := Deps{
		Stores:  stores,
		Keys:    keys,
		Rand:    crypto.NewSecureRand(),
		Clock:   DefaultClock(),
		Limiter: nil, // plug later if needed
		CSRF:    csrf,
		Logger:  logger,
		Mailer:  nil, // plug later if needed
		KV:      memstore.NewKV(),
	}
	// cache for later access
	depsOnce.Do(func() { depsCached = deps })
	return deps
}

var (
	depsOnce   sync.Once
	depsCached Deps
)

// BuildDepsCached returns a cached set of deps after initial construction.
func BuildDepsCached() Deps { return depsCached }

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

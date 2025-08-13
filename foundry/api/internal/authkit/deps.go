package authkit

import (
	"context"
	"time"

	libauth "github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"
	akcrypto "github.com/catalystgo/catalyst-forge/lib/foundry/authkit/crypto"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/store/gormstore"
	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	apihttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	apicfg "github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"gorm.io/gorm"
)

// BuildDeps assembles AuthKit dependencies from API config and infrastructure.
func BuildDeps(ctx context.Context, c *apicfg.Config, dbStore repodb.Store, gdb *gorm.DB, keys akcrypto.KeyManager, logger libauth.Logger) libauth.Deps {
	cookieCfg := apihttp.DefaultCookieConfig()
	csrf := apihttp.NewDoubleSubmitCSRF(cookieCfg, 24*time.Hour)

	stores := libauth.Stores{
		Users:         gormstore.NewUserStore(gdb),
		Credentials:   gormstore.NewCredentialStore(gdb),
		Invites:       gormstore.NewInviteStore(gdb),
		RecoveryCodes: gormstore.NewRecoveryCodeStore(gdb),
		Refresh:       gormstore.NewRefreshStore(gdb),
		Audit:         gormstore.NewAuditStore(gdb),
		// Use in-memory challenge store for WebAuthn ceremonies unless a DB-backed implementation is added
		Challenges: nil,
	}

	deps := libauth.Deps{
		Stores:  stores,
		Keys:    keys,
		Rand:    akcrypto.NewSecureRand(),
		Clock:   libauth.DefaultClock(),
		Limiter: nil, // plug later if needed
		CSRF:    csrf,
		Logger:  logger,
		Mailer:  nil, // plug later if needed
		KV:      nil,
	}
	return deps
}

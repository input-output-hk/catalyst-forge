package authkit

import (
	"context"
	"sync"
	"time"

	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	apihttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	libauth "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/authkit"
	akcrypto "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store/gormstore"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store/memstore"
	apicfg "github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"gorm.io/gorm"
)

// BuildDeps assembles AuthKit dependencies from API config and infrastructure.
func BuildDeps(ctx context.Context, c *apicfg.Config, dbStore repodb.Store, gdb *gorm.DB, keys akcrypto.KeyManager, logger libauth.Logger) libauth.Deps {
	cookieCfg := apihttp.DefaultCookieConfig()
	csrf := apihttp.NewDoubleSubmitCSRF(cookieCfg, 24*time.Hour)

	stores := libauth.Stores{
		Users:          gormstore.NewUserStore(gdb),
		Credentials:    gormstore.NewCredentialStore(gdb),
		Invites:        gormstore.NewInviteStore(gdb),
		RecoveryCodes:  gormstore.NewRecoveryCodeStore(gdb),
		Refresh:        gormstore.NewRefreshStore(gdb),
		Audit:          gormstore.NewAuditStore(gdb),
		Challenges:     memstore.NewChallengeStore(),
		Bootstrap:      gormstore.NewBootstrapTokenStore(gdb),
		GithubPolicies: gormstore.NewGithubPolicyStore(gdb),
		DeviceLinks:    gormstore.NewDeviceLinkStore(gdb),
		Devices:        gormstore.NewDeviceStore(gdb),
	}

	// Keys: prefer provided key manager; if nil, create a manager with a default key
	if keys == nil {
		keys = akcrypto.NewES256KeyManager()
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
		KV:      memstore.NewKV(),
	}
	// cache for later access
	depsOnce.Do(func() { depsCached = deps })
	return deps
}

var (
	depsOnce   sync.Once
	depsCached libauth.Deps
)

// BuildDepsCached returns a cached set of deps after initial construction.
func BuildDepsCached() libauth.Deps { return depsCached }

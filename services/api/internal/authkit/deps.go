package authkit

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"os"
	"sync"
	"time"

	repodb "github.com/catalystgo/catalyst-forge/lib/foundry/db"
	apihttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	libauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	akcrypto "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store/gormstore"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store/memstore"
	apicfg "github.com/input-output-hk/catalyst-forge/services/api/internal/config"
	"gorm.io/gorm"
)

// BuildDeps assembles AuthKit dependencies from API config and infrastructure.
func BuildDeps(ctx context.Context, c *apicfg.Config, dbStore repodb.Store, gdb *gorm.DB, keys akcrypto.KeyManager, logger libauth.Logger) libauth.Deps {
	cookieCfg := apihttp.DefaultCookieConfig()

	// Build CSRF with optional persistent secret
	var csrf apihttp.CSRF = nil
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
		csrf = apihttp.NewDoubleSubmitCSRFWithSecret(cookieCfg, 24*time.Hour, secret)
	} else {
		csrf = apihttp.NewDoubleSubmitCSRF(cookieCfg, 24*time.Hour)
	}

	stores := libauth.Stores{
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
		km := akcrypto.NewStrictES256KeyManager()
		if c != nil {
			kid := c.Auth.SigningKeyKID
			if kid == "" {
				kid = "default"
			}
			if s, ok := km.(*akcrypto.StrictES256KeyManager); ok {
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

package routes

import (
	httpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	authpolicy "github.com/input-output-hk/catalyst-forge/services/api/internal/auth/policy"
	apiauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	libauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
)

// RegisterAuthKit mounts AuthKit routes and auxiliary API-owned auth endpoints.
func RegisterAuthKit(r *gin.Engine, m libauth.Manager, cfg libauth.Config) {
	r.Use(m.Authenticate())
	r.Use(m.EnforcePolicies(authpolicy.BuildRegistry()))

	cookieCfg := httpkit.DefaultCookieConfig()
	deps := apiauth.BuildDepsCached()
	tokenSvc := akservice.NewTokenService(deps.Keys, deps.Rand, cfg.Origin, cfg.AccessTokenTTL)
	refreshSvc := akservice.NewRefreshService(akservice.RefreshServiceConfig{Store: deps.Stores.Refresh, UserStore: deps.Stores.Users, TokenSvc: tokenSvc, Rand: deps.Rand, TTL: cfg.RefreshTokenTTL, AuditStore: deps.Stores.Audit})
	waSvc, _ := akservice.NewWebAuthnService(akservice.WebAuthnConfig{RPDisplayName: cfg.RPName, RPID: cfg.RPID, RPOrigins: []string{cfg.Origin}, Users: deps.Stores.Users, Credentials: deps.Stores.Credentials, Challenges: deps.Stores.Challenges, Rand: deps.Rand, AdminAAGUIDAllowlist: cfg.AdminAAGUIDAllowlist, ChallengeTTL: cfg.ChallengeTTL})

	bindAuthKitRoutes(
		r,
		refreshSvc,
		deps.CSRF,
		cookieCfg,
		waSvc,
		tokenSvc,
		cfg.RefreshTokenTTL,
		cfg.StepUpTTL,
		cfg.BootstrapToken,
		cfg.Origin,
		cfg.GitHub.Enabled,
		cfg.GitHub.Issuer,
		cfg.GitHub.Audiences,
		cfg.GitHub.ExchangeTTL,
	)
}

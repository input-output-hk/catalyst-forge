package routes

import (
	httpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	apiauth "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit"
	libauth "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/authkit"
	rbacpolicy "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/policy"
	akservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/service"
)

// RegisterAuthKit mounts AuthKit routes and auxiliary API-owned auth endpoints.
func RegisterAuthKit(r *gin.Engine, m libauth.Manager, cfg libauth.Config) {
	// Apply global middlewares
	r.Use(m.Authenticate())
	// Prefer centralized RBAC policy registry
	r.Use(m.EnforcePolicies(rbacpolicy.BuildRegistry()))

	// Bind API-owned routes to concrete handlers (refresh/logout/me/session this pass)
	cookieCfg := httpkit.DefaultCookieConfig()
	// Use cached deps constructed during setupAuthKit to ensure non-nil DB handles
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
	// JWKS is registered once in setupAuthKit to avoid duplicate route panic

	// Me and session registered via bindAuthKitRoutes
}

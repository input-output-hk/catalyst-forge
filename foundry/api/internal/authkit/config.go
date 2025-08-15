package authkit

import (
	"net/http"
	"time"

	libauth "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/authkit"
	apicfg "github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
)

// BuildConfig converts API config to AuthKit config with sensible defaults.
func BuildConfig(c *apicfg.Config) libauth.Config {
	cfg := libauth.DefaultConfig()

	// WebAuthn
	if c.Auth.RPName != "" {
		cfg.RPName = c.Auth.RPName
	} else {
		cfg.RPName = "Foundry Platform"
	}
	cfg.RPID = hostFromBaseURL(c.Server.PublicBaseURL)
	cfg.Origin = c.Server.PublicBaseURL
	if c.Auth.ChallengeTTL > 0 {
		cfg.ChallengeTTL = c.Auth.ChallengeTTL
	}
	if c.Auth.RequireUV {
		cfg.RequireUV = true
	}

	// Tokens
	cfg.AccessTokenTTL = c.Auth.AccessTTL
	cfg.RefreshTokenTTL = c.Auth.RefreshTTL
	if c.Auth.StepUpTTL > 0 {
		cfg.StepUpTTL = c.Auth.StepUpTTL
	} else {
		cfg.StepUpTTL = 5 * time.Minute
	}

	// Cookies
	if c.Auth.RefreshCookieName != "" {
		cfg.RefreshCookieName = c.Auth.RefreshCookieName
	}
	if c.Auth.RefreshCookieSecure {
		cfg.SecureCookies = true
	}
	cfg.SameSite = sameSiteFromString(c.Server.CookieSameSite)

	// Admin policy and misc
	cfg.AdminAAGUIDAllowlist = c.Auth.AdminAAGUIDAllowlist
	cfg.RateEnabled = c.Auth.RateEnabled
	if c.Auth.JWKSRoute {
		cfg.JWKSRoute = true
	}
	// Bootstrap token (picked from env/config via Viper)
	cfg.BootstrapToken = c.Auth.BootstrapToken
	cfg.InviteDefaultTTL = c.Auth.InviteTTL
	cfg.InviteMaxAttempts = c.Auth.InviteMaxAttempts

	// GitHub OIDC
	cfg.GitHub.Enabled = c.Auth.GitHub.Enabled
	if c.Auth.GitHub.Issuer != "" {
		cfg.GitHub.Issuer = c.Auth.GitHub.Issuer
	}
	cfg.GitHub.Audiences = c.Auth.GitHub.Audiences
	if c.Auth.GitHub.JWKSCacheTTL > 0 {
		cfg.GitHub.JWKSCacheTTL = c.Auth.GitHub.JWKSCacheTTL
	}
	if c.Auth.GitHub.ExchangeTTL > 0 {
		cfg.GitHub.ExchangeTTL = c.Auth.GitHub.ExchangeTTL
	}
	return cfg
}

func sameSiteFromString(s string) http.SameSite {
	switch s {
	case "Lax", "lax":
		return http.SameSiteLaxMode
	case "None", "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}

func hostFromBaseURL(u string) string {
	// minimal extraction; leave robust parsing for later if needed
	// expects https://host[:port]
	if u == "" {
		return ""
	}
	// strip scheme
	if idx := indexAfter(u, "://"); idx > 0 {
		u = u[idx:]
	}
	// cut path
	if slash := indexOf(u, '/'); slash >= 0 {
		u = u[:slash]
	}
	// cut port
	if colon := indexOf(u, ':'); colon >= 0 {
		u = u[:colon]
	}
	return u
}

func indexAfter(s, sub string) int {
	if i := indexOf(s, sub[0]); i >= 0 {
		if len(s) >= i+len(sub) && s[i:i+len(sub)] == sub {
			return i + len(sub)
		}
	}
	return -1
}

func indexOf(s string, ch byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			return i
		}
	}
	return -1
}

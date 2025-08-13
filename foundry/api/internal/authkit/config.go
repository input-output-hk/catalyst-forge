package authkit

import (
	"net/http"
	"time"

	libauth "github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"
	apicfg "github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
)

// BuildConfig converts API config to AuthKit config with sensible defaults.
func BuildConfig(c *apicfg.Config) libauth.Config {
	cfg := libauth.DefaultConfig()

	// WebAuthn
	cfg.RPName = "Foundry Platform"
	cfg.RPID = hostFromBaseURL(c.Server.PublicBaseURL)
	cfg.Origin = c.Server.PublicBaseURL
	cfg.ChallengeTTL = 5 * time.Minute
	cfg.RequireUV = true

	// Tokens
	cfg.AccessTokenTTL = c.Auth.AccessTTL
	cfg.RefreshTokenTTL = c.Auth.RefreshTTL
	cfg.StepUpTTL = 5 * time.Minute

	// Cookies
	cfg.RefreshCookieName = "__Host-refresh_token"
	cfg.SecureCookies = true
	cfg.SameSite = sameSiteFromString(c.Server.CookieSameSite)

	// Admin policy and misc
	cfg.AdminAAGUIDAllowlist = []string{}
	cfg.RateEnabled = true
	cfg.JWKSRoute = true
	cfg.InviteDefaultTTL = c.Auth.InviteTTL
	cfg.InviteMaxAttempts = c.Auth.InviteMaxAttempts
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

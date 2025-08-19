package authkit

import (
	"net/http"
	"time"

	apicfg "github.com/input-output-hk/catalyst-forge/services/api/internal/config"
)

// Config holds all configuration for the authentication system.
type Config struct {
	// WebAuthn configuration
	RPName       string        // Relying party display name (e.g., "Foundry Platform")
	RPID         string        // Relying party ID (e.g., "foundry.example.com")
	Origin       string        // Expected origin for WebAuthn ceremonies (e.g., "https://foundry.example.com")
	ChallengeTTL time.Duration // TTL for WebAuthn challenges (default: 5m)
	RequireUV    bool          // Require user verification (default: true)

	// Token configuration
	AccessTokenTTL  time.Duration // JWT access token lifetime (default: 30m)
	RefreshTokenTTL time.Duration // Refresh token lifetime (default: 30d)
	StepUpTTL       time.Duration // Step-up authentication validity window (default: 5m)

	// Cookie configuration
	RefreshCookieName string        // Name for refresh token cookie (default: "__Host-refresh_token")
	SecureCookies     bool          // HTTPS-only cookies (default: true)
	SameSite          http.SameSite // Cookie SameSite attribute (default: SameSiteStrict)

	// Admin policy
	AdminAAGUIDAllowlist []string // List of allowed AAGUIDs for admin users (hardware keys)

	// Rate limiting
	RateEnabled bool // Enable rate limiting (default: false, requires limiter in Deps)

	// JWKS exposure
	JWKSRoute bool // Expose GET /.well-known/jwks.json endpoint (default: true)

	// Miscellaneous
	InviteDefaultTTL  time.Duration // Default time-to-live for invites (default: 72h)
	InviteMaxAttempts int           // Maximum failed attempts before invite is locked (default: 5)

	// Bootstrap
	// BootstrapToken enables one-time administrator bootstrap when set.
	// The token must be exactly 32 characters and will be recorded on first use
	// to prevent replay.
	BootstrapToken string

	// GitHub OIDC configuration
	GitHub struct {
		Enabled      bool
		Issuer       string        // default: https://token.actions.githubusercontent.com
		Audiences    []string      // allow-list for aud claim
		JWKSCacheTTL time.Duration // default: 15m
		ExchangeTTL  time.Duration // default: 15m
	}
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ChallengeTTL:      5 * time.Minute,
		RequireUV:         true,
		AccessTokenTTL:    30 * time.Minute,
		RefreshTokenTTL:   30 * 24 * time.Hour,
		StepUpTTL:         5 * time.Minute,
		RefreshCookieName: "__Host-refresh_token",
		SecureCookies:     true,
		SameSite:          http.SameSiteStrictMode,
		RateEnabled:       false,
		JWKSRoute:         true,
		InviteDefaultTTL:  72 * time.Hour,
		InviteMaxAttempts: 5,
		GitHub: struct {
			Enabled      bool
			Issuer       string
			Audiences    []string
			JWKSCacheTTL time.Duration
			ExchangeTTL  time.Duration
		}{
			Enabled:      false,
			Issuer:       "https://token.actions.githubusercontent.com",
			Audiences:    nil,
			JWKSCacheTTL: 15 * time.Minute,
			ExchangeTTL:  15 * time.Minute,
		},
	}
}

// BuildConfig converts API config to AuthKit config with sensible defaults.
func BuildConfig(c *apicfg.Config) Config {
	cfg := DefaultConfig()

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

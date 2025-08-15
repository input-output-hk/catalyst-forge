package authkit

import (
	"net/http"
	"time"
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

package auth

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
)

// CookieManager handles secure cookie operations for device-keypair authentication.
type CookieManager struct {
	config *config.AuthConfig
}

// NewCookieManager creates a new cookie manager with the provided configuration.
func NewCookieManager(config *config.AuthConfig) *CookieManager {
	return &CookieManager{
		config: config,
	}
}

// RefreshTokenCookie represents the parsed components of a refresh token cookie.
type RefreshTokenCookie struct {
	JTI    uuid.UUID // JSON Token Identifier
	Secret string    // Base64URL-encoded secret
}

// SetRefreshTokenCookie sets a secure refresh token cookie using the format:
// rt = base64url(jti) + "." + base64url(secret).
func (cm *CookieManager) SetRefreshTokenCookie(c *gin.Context, jti uuid.UUID, secret string, ttl time.Duration) {
	// Encode components using base64url (no padding)
	jtiEncoded := base64.RawURLEncoding.EncodeToString([]byte(jti.String()))
	secretEncoded := base64.RawURLEncoding.EncodeToString([]byte(secret))

	// Create cookie value in the required format
	cookieValue := jtiEncoded + "." + secretEncoded

	// Set cookie with secure flags
	cm.setSecureCookie(c, cm.config.RefreshCookieName, cookieValue, ttl, "/auth")
}

// ParseRefreshTokenCookie parses and validates the refresh token cookie format.
func (cm *CookieManager) ParseRefreshTokenCookie(c *gin.Context) (*RefreshTokenCookie, error) {
	// Get cookie value
	cookieValue, err := c.Cookie(cm.config.RefreshCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return nil, errors.New("refresh token cookie not found")
		}
		return nil, err
	}

	return cm.ParseRefreshTokenValue(cookieValue)
}

// ParseRefreshTokenValue parses a refresh token cookie value string.
func (cm *CookieManager) ParseRefreshTokenValue(cookieValue string) (*RefreshTokenCookie, error) {
	if cookieValue == "" {
		return nil, errors.New("cookie value is empty")
	}

	// Split on the dot separator
	parts := strings.SplitN(cookieValue, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid cookie format: expected 'jti.secret'")
	}

	// Decode JTI
	jtiBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("invalid JTI encoding")
	}

	jti, err := uuid.Parse(string(jtiBytes))
	if err != nil {
		return nil, errors.New("invalid JTI format")
	}

	// Decode secret
	secretBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("invalid secret encoding")
	}

	return &RefreshTokenCookie{
		JTI:    jti,
		Secret: string(secretBytes),
	}, nil
}

// ClearRefreshTokenCookie clears the refresh token cookie for logout.
func (cm *CookieManager) ClearRefreshTokenCookie(c *gin.Context) {
	cm.clearSecureCookie(c, cm.config.RefreshCookieName, "/auth")
}

// SetAccessTokenCookie sets an access token cookie (for browser clients).
func (cm *CookieManager) SetAccessTokenCookie(c *gin.Context, jwt string, ttl time.Duration) {
	// Access tokens use shorter path scope and stricter settings
	cm.setSecureCookie(c, "cforge_at", jwt, ttl, "/")
}

// ClearAccessTokenCookie clears the access token cookie.
func (cm *CookieManager) ClearAccessTokenCookie(c *gin.Context) {
	cm.clearSecureCookie(c, "cforge_at", "/")
}

// setSecureCookie sets a cookie with proper security flags based on configuration.
func (cm *CookieManager) setSecureCookie(c *gin.Context, name, value string, ttl time.Duration, path string) {
	// Determine security flags
	secure := cm.shouldUseSecureFlag(c)
	domain := cm.getCookieDomain(c)

	// Set SameSite policy to Lax for auth cookies (allows navigation from external sites)
	c.SetSameSite(http.SameSiteLaxMode)

	// Set cookie with security flags
	// HttpOnly=true, Secure=auto-detected, SameSite=Lax, Path=specified
	c.SetCookie(name, value, int(ttl.Seconds()), path, domain, secure, true)
}

// clearSecureCookie clears a cookie by setting it to expire in the past.
func (cm *CookieManager) clearSecureCookie(c *gin.Context, name, path string) {
	domain := cm.getCookieDomain(c)
	secure := cm.shouldUseSecureFlag(c)

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, "", -1, path, domain, secure, true)
}

// shouldUseSecureFlag determines if cookies should have the Secure flag.
func (cm *CookieManager) shouldUseSecureFlag(c *gin.Context) bool {
	// If explicitly configured, use that setting
	if cm.config.RefreshCookieSecure {
		return true
	}

	// Auto-detect based on the request scheme
	if c.Request.TLS != nil {
		return true
	}

	// Check forwarded headers (for proxies)
	if proto := c.GetHeader("X-Forwarded-Proto"); proto == "https" {
		return true
	}

	if scheme := c.GetHeader("X-Forwarded-Scheme"); scheme == "https" {
		return true
	}

	// Default to false for local development
	return false
}

// getCookieDomain returns the domain to use for cookies based on configuration.
func (cm *CookieManager) getCookieDomain(c *gin.Context) string {
	// If explicitly configured, use that domain
	if cm.config.RefreshCookieDomain != "" {
		return cm.config.RefreshCookieDomain
	}

	// Auto-detect from request host
	host := c.Request.Host
	if host == "" {
		return ""
	}

	// Strip port if present
	if colonIndex := strings.LastIndex(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	// Don't set domain for localhost/127.0.0.1 (browser will handle it correctly)
	if host == "localhost" || strings.HasPrefix(host, "127.") || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") {
		return ""
	}

	return host
}

// ValidateRefreshTokenCookieFormat validates that a cookie value matches the expected format
// without parsing the actual values (for format validation).
func (cm *CookieManager) ValidateRefreshTokenCookieFormat(cookieValue string) error {
	if cookieValue == "" {
		return errors.New("cookie value is empty")
	}

	parts := strings.SplitN(cookieValue, ".", 2)
	if len(parts) != 2 {
		return errors.New("invalid format: expected 'jti.secret'")
	}

	if parts[0] == "" || parts[1] == "" {
		return errors.New("invalid format: both JTI and secret must be non-empty")
	}

	// Validate that parts are valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil {
		return errors.New("invalid JTI encoding")
	}

	if _, err := base64.RawURLEncoding.DecodeString(parts[1]); err != nil {
		return errors.New("invalid secret encoding")
	}

	return nil
}

// GetRefreshTokenCookieName returns the configured refresh token cookie name.
func (cm *CookieManager) GetRefreshTokenCookieName() string {
	return cm.config.RefreshCookieName
}

package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/rate"
)

// AuthRateLimitMiddleware provides rate limiting for authentication endpoints per user.
type AuthRateLimitMiddleware struct {
	limiter    rate.Limiter
	authConfig *config.AuthConfig
	logger     *slog.Logger
}

// NewAuthRateLimitMiddleware creates a new auth rate limiting middleware.
func NewAuthRateLimitMiddleware(limiter rate.Limiter, authConfig *config.AuthConfig, logger *slog.Logger) *AuthRateLimitMiddleware {
	return &AuthRateLimitMiddleware{
		limiter:    limiter,
		authConfig: authConfig,
		logger:     logger,
	}
}

// Handle applies rate limiting based on user identification.
func (m *AuthRateLimitMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow disabling rate limiting in test environments
		if os.Getenv("AUTH_RATELIMIT_DISABLE") == "1" {
			c.Next()
			return
		}
		userKey := m.getUserKey(c)
		if userKey == "" {
			// No user identification available, allow but log
			m.logger.Debug("Auth rate limit: no user identification available", "path", c.Request.URL.Path)
			c.Next()
			return
		}

		// Check rate limit
		allowed, err := m.limiter.Allow(
			c.Request.Context(),
			fmt.Sprintf("auth:%s", userKey),
			m.authConfig.AuthRateLimitBurst,
			m.authConfig.AuthRateLimitWindow,
		)

		if err != nil {
			m.logger.Error("Auth rate limit check failed", "error", err, "user", userKey)
			// On error, allow the request but log the issue
			c.Next()
			return
		}

		if !allowed {
			m.logger.Warn("Auth rate limit exceeded",
				"user", userKey,
				"path", c.Request.URL.Path,
				"burst", m.authConfig.AuthRateLimitBurst,
				"window", m.authConfig.AuthRateLimitWindow)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded. Please try again later.",
				"retry_after": int(m.authConfig.AuthRateLimitWindow.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getUserKey attempts to extract a unique user identifier from the request.
func (m *AuthRateLimitMiddleware) getUserKey(c *gin.Context) string {
	// Try to get user from authentication context (for authenticated endpoints)
	if user, exists := c.Get("user"); exists {
		if authUser, ok := user.(*AuthenticatedUser); ok && authUser.ID != "" {
			return fmt.Sprintf("user:%s", authUser.ID)
		}
	}

	path := c.Request.URL.Path

	// For device registration endpoints, we can rate limit by invite token
	if path == "/auth/devices/init" || path == "/auth/devices/register" {
		return m.getDeviceRegistrationUserKey(c)
	}

	// For refresh endpoint, try to extract device ID from headers
	if path == "/auth/refresh" {
		return m.getRefreshTokenUserKey(c)
	}

	// For other unauthenticated endpoints, use a global rate limit key
	// This provides some protection even without user identification
	return "anonymous"
}

// getDeviceRegistrationUserKey extracts user key for device registration endpoints.
func (m *AuthRateLimitMiddleware) getDeviceRegistrationUserKey(c *gin.Context) string {
	// For device registration endpoints, we can rate limit by invite token in the session
	// This provides user-specific rate limiting during registration
	if inviteToken, exists := c.Get("invite_token"); exists {
		if token, ok := inviteToken.(string); ok && token != "" {
			return fmt.Sprintf("invite:%s", token)
		}
	}

	// Fallback to anonymous rate limiting for device registration
	return "device-registration"
}

// getRefreshTokenUserKey extracts user key from refresh token cookie or device headers.
func (m *AuthRateLimitMiddleware) getRefreshTokenUserKey(c *gin.Context) string {
	// Try to extract device ID from X-Device-Id header
	deviceID := c.GetHeader("X-Device-Id")
	if deviceID != "" {
		return fmt.Sprintf("device:%s", deviceID)
	}

	// Fallback to anonymous rate limiting for refresh
	return "token-refresh"
}

// RateLimitKey generates a rate limiting key for a user.
func RateLimitKey(userID string, endpoint string) string {
	return fmt.Sprintf("auth:%s:%s", userID, endpoint)
}

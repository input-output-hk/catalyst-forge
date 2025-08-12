package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
)

// AuthCORSMiddleware provides CORS middleware specifically for /auth/* endpoints.
type AuthCORSMiddleware struct {
	authConfig *config.AuthConfig
	logger     *slog.Logger
}

// NewAuthCORSMiddleware creates a new auth-specific CORS middleware.
func NewAuthCORSMiddleware(authConfig *config.AuthConfig, logger *slog.Logger) *AuthCORSMiddleware {
	return &AuthCORSMiddleware{
		authConfig: authConfig,
		logger:     logger,
	}
}

// Handle applies CORS policy for authentication endpoints.
func (m *AuthCORSMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Parse allowed origins from configuration
		allowedOrigins := m.parseAllowedOrigins()

		// Handle preflight OPTIONS requests
		if c.Request.Method == "OPTIONS" {
			m.handlePreflight(c, origin, allowedOrigins)
			return
		}

		// Validate origin for non-preflight requests
		if origin != "" {
			if !m.isOriginAllowed(origin, allowedOrigins) {
				m.logger.Warn("CORS: Origin not allowed for auth endpoint",
					"origin", origin,
					"path", c.Request.URL.Path,
					"allowed_origins", allowedOrigins)
				c.JSON(http.StatusForbidden, gin.H{"error": "Origin not allowed"})
				c.Abort()
				return
			}

			// Set CORS headers for allowed origins
			m.setCORSHeaders(c, origin)
		}

		c.Next()
	}
}

// handlePreflight handles CORS preflight OPTIONS requests.
func (m *AuthCORSMiddleware) handlePreflight(c *gin.Context, origin string, allowedOrigins []string) {
	if origin == "" {
		// No origin header, reject preflight
		c.Status(http.StatusForbidden)
		c.Abort()
		return
	}

	if !m.isOriginAllowed(origin, allowedOrigins) {
		m.logger.Warn("CORS: Origin not allowed for auth preflight",
			"origin", origin,
			"path", c.Request.URL.Path,
			"allowed_origins", allowedOrigins)
		c.Status(http.StatusForbidden)
		c.Abort()
		return
	}

	// Set preflight headers
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Device-Id, X-Device-Proof")
	c.Header("Access-Control-Max-Age", "3600") // Cache preflight for 1 hour
	c.Header("Vary", "Origin")

	c.Status(http.StatusNoContent)
	c.Abort()
}

// setCORSHeaders sets CORS headers for actual requests.
func (m *AuthCORSMiddleware) setCORSHeaders(c *gin.Context, origin string) {
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Vary", "Origin")
}

// isOriginAllowed checks if the origin is in the allowed list.
func (m *AuthCORSMiddleware) isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	return false
}

// parseAllowedOrigins parses the comma-separated list of allowed origins.
func (m *AuthCORSMiddleware) parseAllowedOrigins() []string {
	if m.authConfig.AllowedWebOrigins == "" {
		return []string{}
	}

	origins := strings.Split(m.authConfig.AllowedWebOrigins, ",")
	var result []string
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

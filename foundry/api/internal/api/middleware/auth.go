package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/auth"
)

// AuthenticatedUser is a struct that contains the user information from the
// authentication middleware
type AuthenticatedUser struct {
	ID          string
	Permissions []auth.Permission
	Claims      *auth.Claims
}

// hasPermissions checks if the user has the required permissions
func (u *AuthenticatedUser) hasPermissions(permissions []auth.Permission) bool {
	for _, required := range permissions {
		if slices.Contains(u.Permissions, required) {
			return true
		}
	}
	return false
}

// AuthMiddleware provides a middleware that validates a user's permissions
type AuthMiddleware struct {
	authManager *auth.AuthManager
	logger      *slog.Logger
}

// ValidatePermissions returns a middleware that validates a user's permissions
func (h *AuthMiddleware) ValidatePermissions(permissions []auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := h.getToken(c)
		if err != nil {
			h.logger.Warn("Invalid token", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
		}

		user, err := h.getUser(token)
		if err != nil {
			h.logger.Warn("Invalid token", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
		}

		if !user.hasPermissions(permissions) {
			h.logger.Warn("Permission denied", "user_id", user.ID, "permissions", permissions)
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Permission denied",
			})
			c.Abort()
		}

		c.Set("user", user)
		c.Next()
	}
}

// getToken extracts the token from the Authorization header
func (h *AuthMiddleware) getToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is required")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("authorization header must start with 'Bearer '")
	}

	return strings.TrimPrefix(authHeader, "Bearer "), nil
}

// getUser validates the token and returns the authenticated user
func (h *AuthMiddleware) getUser(token string) (*AuthenticatedUser, error) {
	claims, err := h.authManager.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	return &AuthenticatedUser{
		ID:          claims.UserID,
		Permissions: claims.Permissions,
		Claims:      claims,
	}, nil
}

// NewAuthMiddleware creates a new AuthMiddlewareHandler
func NewAuthMiddleware(authManager *auth.AuthManager, logger *slog.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authManager: authManager,
		logger:      logger,
	}
}

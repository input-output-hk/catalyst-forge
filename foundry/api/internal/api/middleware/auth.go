package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
    userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
    userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
)

// AuthenticatedUser is a struct that contains the user information from the
// authentication middleware
type AuthenticatedUser struct {
    ID          string
    Permissions []auth.Permission
    Claims      *tokens.AuthClaims
}

// hasPermissions checks if the user has the required permissions
func (u *AuthenticatedUser) hasAllPermissions(permissions []auth.Permission) bool {
	for _, required := range permissions {
		if !slices.Contains(u.Permissions, required) {
			return false
		}
	}
	return true
}

func (u *AuthenticatedUser) hasAnyPermissions(permissions []auth.Permission) bool {
	for _, required := range permissions {
		if slices.Contains(u.Permissions, required) {
			return true
		}
	}
	return false
}

// AuthMiddleware provides a middleware that validates a user's permissions
type AuthMiddleware struct {
	jwtManager jwt.JWTManager
	logger     *slog.Logger
    userService userservice.UserService
    revokedRepo userrepo.RevokedJTIRepository
}

// ValidatePermissions returns a middleware that validates a user's permissions
// ValidatePermissions enforces RequireAll (AND) by default
func (h *AuthMiddleware) ValidatePermissions(permissions []auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := h.getToken(c)
		if err != nil {
			h.logger.Warn("Invalid token", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

        user, err := h.getUser(token)
		if err != nil {
        if err := h.validateClaims(user); err != nil {
            h.logger.Warn("Token rejected", "error", err)
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }
			h.logger.Warn("Invalid token", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		if !user.hasAllPermissions(permissions) {
			h.logger.Warn("Permission denied", "user_id", user.ID, "permissions", permissions)
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Permission denied",
			})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

// RequireAny returns a middleware that enforces OR semantics across provided permissions
func (h *AuthMiddleware) RequireAny(permissions []auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := h.getToken(c)
		if err != nil {
			h.logger.Warn("Invalid token", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

        user, err := h.getUser(token)
		if err != nil {
        if err := h.validateClaims(user); err != nil {
            h.logger.Warn("Token rejected", "error", err)
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }
			h.logger.Warn("Invalid token", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if !user.hasAnyPermissions(permissions) {
			h.logger.Warn("Permission denied", "user_id", user.ID, "permissions", permissions)
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

// ValidateAnyCertificatePermission returns a middleware that validates the user has any certificate signing permission
func (h *AuthMiddleware) ValidateAnyCertificatePermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := h.getToken(c)
		if err != nil {
			h.logger.Warn("Invalid token", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

        user, err := h.getUser(token)
		if err != nil {
        if err := h.validateClaims(user); err != nil {
            h.logger.Warn("Token rejected", "error", err)
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }
			h.logger.Warn("Invalid token", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		if !tokens.HasAnyCertificateSignPermission(user.Claims) {
			h.logger.Warn("Permission denied", "user_id", user.ID, "reason", "no certificate signing permissions")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Permission denied: no certificate signing permissions",
			})
			c.Abort()
			return
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
	claims, err := tokens.VerifyAuthToken(h.jwtManager, token)
	if err != nil {
		return nil, err
	}

	return &AuthenticatedUser{
		ID:          claims.Subject,
		Permissions: claims.Permissions,
		Claims:      claims,
	}, nil
}

func (h *AuthMiddleware) validateClaims(user *AuthenticatedUser) error {
    claims := user.Claims
    // Issuer check
    if claims.Issuer != h.jwtManager.Issuer() {
        return fmt.Errorf("issuer mismatch")
    }
    // Audience check (at least one match)
    want := h.jwtManager.DefaultAudiences()
    matched := false
    for _, a := range claims.Audience {
        for _, w := range want {
            if a == w {
                matched = true
                break
            }
        }
        if matched {
            break
        }
    }
    if !matched {
        return fmt.Errorf("audience mismatch")
    }
    // JTI denylist
    if claims.ID != "" {
        revoked, err := h.revokedRepo.IsRevoked(claims.ID)
        if err != nil {
            return err
        }
        if revoked {
            return fmt.Errorf("token revoked")
        }
    }
    // user_ver freshness
    u, err := h.userService.GetUserByEmail(claims.Subject)
    if err != nil {
        return fmt.Errorf("user lookup failed")
    }
    if u.UserVer > claims.UserVer {
        return fmt.Errorf("stale token")
    }
    return nil
}

// NewAuthMiddleware creates a new AuthMiddlewareHandler
func NewAuthMiddleware(jwtManager jwt.JWTManager, logger *slog.Logger, userService userservice.UserService, revokedRepo userrepo.RevokedJTIRepository) *AuthMiddleware {
	return &AuthMiddleware{
        jwtManager:  jwtManager,
        logger:      logger,
        userService: userService,
        revokedRepo: revokedRepo,
	}
}

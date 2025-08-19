package authkit

import (
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// contextKey is the key used to store AuthContext in gin.Context
const contextKey = "auth_context"

// AuthContext contains the authenticated user's information.
type AuthContext struct {
	UserID           uuid.UUID  // User's unique identifier
	Email            string     // User's email address
	FullName         string     // User's display name
	Roles            []string   // User's roles
	Permissions      []string   // User's permissions (optional, derived from roles)
	SessionVersion   int64      // Session version for invalidation
	StepUpValidUntil time.Time  // If set, user recently performed step-up authentication
	TokenID          string     // JWT ID (jti) for tracking
	AMR              []string   // Authentication Methods References (e.g., ["webauthn"], ["device_link"])
	DeviceID         *uuid.UUID // Device ID for CLI sessions (nil for web sessions)
}

// From retrieves the AuthContext from a gin.Context.
//
// Returns the context and true if found, or an empty context and false if not found.
func From(c *gin.Context) (AuthContext, bool) {
	val, exists := c.Get(contextKey)
	if !exists {
		return AuthContext{}, false
	}

	ctx, ok := val.(AuthContext)
	return ctx, ok
}

// Must retrieves the AuthContext from a gin.Context.
//
// Panics if the context is not found - use only in handlers that are guaranteed to have auth.
func Must(c *gin.Context) AuthContext {
	ctx, ok := From(c)
	if !ok {
		panic("AuthContext not found in gin.Context - ensure Authenticate middleware is applied")
	}
	return ctx
}

// Set stores an AuthContext in a gin.Context.
func (ac AuthContext) Set(c *gin.Context) {
	c.Set(contextKey, ac)
}

// IsAuthenticated returns true if the user is authenticated.
func (ac AuthContext) IsAuthenticated() bool {
	return ac.UserID != uuid.Nil
}

// HasRole checks if the user has a specific role.
func (ac AuthContext) HasRole(role string) bool {
	return slices.Contains(ac.Roles, role)
}

// HasAnyRole checks if the user has any of the specified roles.
func (ac AuthContext) HasAnyRole(roles []string) bool {
	for _, role := range roles {
		if slices.Contains(ac.Roles, role) {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has a specific permission.
func (ac AuthContext) HasPermission(permission string) bool {
	return slices.Contains(ac.Permissions, permission)
}

// HasAnyPermission checks if the user has any of the specified permissions.
func (ac AuthContext) HasAnyPermission(permissions []string) bool {
	for _, permission := range permissions {
		if slices.Contains(ac.Permissions, permission) {
			return true
		}
	}
	return false
}

// RequiresStepUp checks if step-up authentication is required.
func (ac AuthContext) RequiresStepUp(now time.Time) bool {
	return ac.StepUpValidUntil.IsZero() || now.After(ac.StepUpValidUntil)
}

// SetContext stores an AuthContext in a gin.Context.
//
// This is a convenience function for testing and middleware.
func SetContext(c *gin.Context, ctx AuthContext) {
	c.Set(contextKey, ctx)
}

package middleware

import (
    "net/http"
    "time"

    "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/authkit"
    basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
    "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/service"
    "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

// Authenticator provides JWT authentication middleware.
type Authenticator struct {
	tokenService service.TokenService
	userStore    store.UserStore
	issuer       string
}

// NewAuthenticator creates a new authentication middleware.
func NewAuthenticator(tokenService service.TokenService, userStore store.UserStore, issuer string) *Authenticator {
	return &Authenticator{
		tokenService: tokenService,
		userStore:    userStore,
		issuer:       issuer,
	}
}

// Authenticate is middleware that validates JWT tokens and injects AuthContext.
//
// This middleware:
//   - Extracts the bearer token from the Authorization header
//   - Validates the JWT signature and claims
//   - Verifies the session version matches the user's current version
//   - Creates and injects an AuthContext into the request context
//
// If authentication fails, the request continues without auth context.
// Use RequireAuth to enforce authentication.
func (a *Authenticator) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract bearer token
        token, err := basehttpkit.GetBearerToken(c.Request)
		if err != nil || token == "" {
			// No token provided, continue without auth
			c.Next()
			return
		}

		// Verify the JWT
		claims, err := a.tokenService.ParseAccess(c.Request.Context(), token)
		if err != nil {
			// Invalid token, continue without auth
			c.Next()
			return
		}

		// Validate issuer
		if claims.Issuer != a.issuer {
			c.Next()
			return
		}

		// Parse user ID
		userID, err := uuid.Parse(claims.Sub)
		if err != nil {
			c.Next()
			return
		}

		// Get user to verify session version
		user, err := a.userStore.GetByID(c.Request.Context(), userID)
		if err != nil || user == nil {
			c.Next()
			return
		}

		// Verify session version matches
		if user.SessionVersion != claims.SessionVersion {
			// Session has been invalidated
			c.Next()
			return
		}

		// Parse step-up expiry if present
		var stepUpValidUntil time.Time
		if claims.StepUpUntil > 0 {
			stepUpValidUntil = time.Unix(claims.StepUpUntil, 0)
		}

		// Parse device ID if present (for CLI sessions)
		var deviceID *uuid.UUID
		if claims.DeviceID != "" {
			if did, err := uuid.Parse(claims.DeviceID); err == nil {
				deviceID = &did
			}
		}

		// Create auth context
		authCtx := authkit.AuthContext{
			UserID:           userID,
			Email:            user.Email,
			Roles:            user.Roles,
			Permissions:      user.Permissions,
			SessionVersion:   user.SessionVersion,
			StepUpValidUntil: stepUpValidUntil,
			TokenID:          claims.JTI,
			AMR:              claims.AMR,
			DeviceID:         deviceID,
		}

		// Inject into gin context
		authCtx.Set(c)

		c.Next()
	}
}

// RequireAuth ensures the request is authenticated.
//
// Returns 401 if no valid authentication context is found.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() {
            basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "unauthorized", "Authentication required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireStepUp ensures the user has recently performed step-up authentication.
//
// Returns 428 Precondition Required if step-up is needed.
func RequireStepUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() {
            basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "unauthorized", "Authentication required")
			c.Abort()
			return
		}

		// Check if step-up is still valid
		if ctx.RequiresStepUp(time.Now().UTC()) {
            basehttpkit.ErrorResponse(c.Writer, http.StatusPreconditionRequired, "step_up_required", "Step-up authentication required")
			c.Abort()
			return
		}

		c.Next()
	}
}


// OptionalAuth is an alias for Authenticate that makes the intent clearer.
//
// Use this when authentication is optional but you want to load auth context if available.
func (a *Authenticator) OptionalAuth() gin.HandlerFunc {
	return a.Authenticate()
}

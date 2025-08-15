package middleware

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
    basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
    "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
    "github.com/gin-gonic/gin"
)

// SessionValidator provides session validation middleware.
type SessionValidator struct {
	userStore store.UserStore
	kv        store.KV
}

// NewSessionValidator creates a new session validation middleware.
func NewSessionValidator(userStore store.UserStore, kv store.KV) *SessionValidator {
	return &SessionValidator{
		userStore: userStore,
		kv:        kv,
	}
}

// ValidateSession performs additional session validation checks.
//
// This middleware:
//   - Re-verifies the session version against the database
//   - Checks for account suspension or deletion
//   - Optionally checks for token revocation (if using a revocation list)
//
// This provides an extra layer of security for critical operations.
func (sv *SessionValidator) ValidateSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() {
			// No auth context, skip validation
			c.Next()
			return
		}

		// Get fresh user data
		user, err := sv.userStore.GetByID(c.Request.Context(), ctx.UserID)
		if err != nil || user == nil {
			// User not found or error
            basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "session_invalid", "Session is no longer valid")
			c.Abort()
			return
		}

		// Check if user is suspended
		if user.SuspendedAt != nil {
            basehttpkit.ErrorResponse(c.Writer, http.StatusForbidden, "account_suspended", "Account has been suspended")
			c.Abort()
			return
		}

		// Re-verify session version
		if user.SessionVersion != ctx.SessionVersion {
            basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "session_expired", "Session has been invalidated")
			c.Abort()
			return
		}

		// Check token revocation list if JTI is present
		if ctx.TokenID != "" && sv.kv != nil {
			revoked, err := sv.isTokenRevoked(c.Request.Context(), ctx.TokenID)
			if err == nil && revoked {
                basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "token_revoked", "Token has been revoked")
				c.Abort()
				return
			}
		}

		// Update auth context with fresh user data
		ctx.Email = user.Email
		ctx.Roles = user.Roles
		ctx.Permissions = user.Permissions
		ctx.Set(c)

		c.Next()
	}
}

// isTokenRevoked checks if a token ID is in the revocation list.
func (sv *SessionValidator) isTokenRevoked(ctx context.Context, jti string) (bool, error) {
	key := "revoked:token:" + jti
	val, err := sv.kv.Get(ctx, key)
	if err != nil {
		if err == store.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return len(val) > 0, nil
}

// RevokeToken adds a token to the revocation list.
//
// The token will be revoked until its natural expiry time.
func (sv *SessionValidator) RevokeToken(ctx context.Context, jti string, expiresAt time.Time) error {
	if sv.kv == nil {
		// No KV store configured, can't revoke tokens
		return nil
	}

	key := "revoked:token:" + jti
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		// Token already expired
		return nil
	}

	// Cap TTL to a reasonable maximum (e.g., 24 hours for access tokens)
	// This prevents issues with misissued tokens with far-future expiry
	const maxTTL = 24 * time.Hour
	if ttl > maxTTL {
		ttl = maxTTL
	}

	return sv.kv.Set(ctx, key, []byte("1"), ttl)
}

// TouchSession updates the last activity timestamp for a user.
//
// This can be used to track user activity for audit or session timeout purposes.
func (sv *SessionValidator) TouchSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() || sv.kv == nil {
			// No auth context or KV store not configured, skip
			c.Next()
			return
		}

		// Update last activity in background to avoid blocking the request
		go func() {
			// Create a new context with timeout for the background operation
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Use UTC for timestamp consistency
			key := "session:activity:" + ctx.UserID.String()
			_ = sv.kv.Set(bgCtx, key, []byte(time.Now().UTC().Format(time.RFC3339)), 24*time.Hour)
		}()

		c.Next()
	}
}

// RequireActiveSession ensures the user has been active recently.
//
// This can be used to require re-authentication after a period of inactivity.
func (sv *SessionValidator) RequireActiveSession(maxInactivity time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() {
            basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "unauthorized", "Authentication required")
			c.Abort()
			return
		}

		if sv.kv == nil {
			// No KV store, can't track activity
			c.Next()
			return
		}

		key := "session:activity:" + ctx.UserID.String()
		val, err := sv.kv.Get(c.Request.Context(), key)
		if err != nil {
            if err == store.ErrNotFound {
                // No activity record, treat as inactive
                basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "session_timeout", "Session has timed out due to inactivity")
                c.Abort()
                return
            }
			// Error checking activity, allow the request
			c.Next()
			return
		}

		lastActivity, err := time.Parse(time.RFC3339, string(val))
		if err != nil {
			// Invalid timestamp, allow the request
			c.Next()
			return
		}

		// Use UTC for consistent comparison
            if time.Since(lastActivity.UTC()) > maxInactivity {
                basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "session_timeout", "Session has timed out due to inactivity")
                c.Abort()
                return
            }

		c.Next()
	}
}

// TouchSessionWithID updates the last activity timestamp for a specific session.
//
// This provides per-session activity tracking instead of per-user.
// The sessionID could be a refresh token family ID, JTI, or other stable identifier.
func (sv *SessionValidator) TouchSessionWithID(sessionID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() || sv.kv == nil {
			// No auth context or KV store not configured, skip
			c.Next()
			return
		}

		// Update last activity in background to avoid blocking the request
		go func() {
			// Create a new context with timeout for the background operation
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Use UTC for timestamp consistency
			// Key includes both user ID and session ID for per-session tracking
			key := fmt.Sprintf("session:activity:%s:%s", ctx.UserID.String(), sessionID)
			_ = sv.kv.Set(bgCtx, key, []byte(time.Now().UTC().Format(time.RFC3339)), 24*time.Hour)
		}()

		c.Next()
	}
}

// RequireActiveSessionWithID ensures a specific session has been active recently.
//
// This provides per-session inactivity checking instead of per-user.
func (sv *SessionValidator) RequireActiveSessionWithID(sessionID string, maxInactivity time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := authkit.From(c)
            if !ok || !ctx.IsAuthenticated() {
                basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "unauthorized", "Authentication required")
                c.Abort()
                return
            }

		if sv.kv == nil {
			// No KV store, can't track activity
			c.Next()
			return
		}

		key := fmt.Sprintf("session:activity:%s:%s", ctx.UserID.String(), sessionID)
		val, err := sv.kv.Get(c.Request.Context(), key)
		if err != nil {
                if err == store.ErrNotFound {
                    // No activity record, treat as inactive
                    basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "session_timeout", "Session has timed out due to inactivity")
                    c.Abort()
                    return
                }
			// Error checking activity, allow the request
			c.Next()
			return
		}

		lastActivity, err := time.Parse(time.RFC3339, string(val))
		if err != nil {
			// Invalid timestamp, allow the request
			c.Next()
			return
		}

		// Use UTC for consistent comparison
            if time.Since(lastActivity.UTC()) > maxInactivity {
                basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "session_timeout", "Session has timed out due to inactivity")
                c.Abort()
                return
            }

		c.Next()
	}
}

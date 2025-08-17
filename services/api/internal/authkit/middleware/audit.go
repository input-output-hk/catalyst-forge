package middleware

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	akauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	httpkit "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/httpkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

// AuditLogger provides audit logging middleware.
type AuditLogger struct {
	auditStore   store.AuditStore
	refreshStore store.RefreshStore
}

// NewAuditLogger creates a new audit logging middleware.
func NewAuditLogger(auditStore store.AuditStore) *AuditLogger {
	return &AuditLogger{
		auditStore: auditStore,
	}
}

// WithRefreshStore allows resolving actor for refresh events when auth context is missing.
func (al *AuditLogger) WithRefreshStore(refresh store.RefreshStore) *AuditLogger {
	al.refreshStore = refresh
	return al
}

// LogAuthEvents creates middleware that logs authentication-related events.
func (al *AuditLogger) LogAuthEvents() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Capture start time (UTC for consistency)
		startTime := time.Now().UTC()

		// Capture request body for auth endpoints (but not passwords)
		var requestBody []byte
		if isAuthEndpoint(c.Request.URL.Path) && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Create response writer wrapper to capture status
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			statusCode:     http.StatusOK,
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Get final status code (prefer Gin's Status() if set)
		statusCode := writer.statusCode
		if c.Writer.Status() > 0 {
			statusCode = c.Writer.Status()
		}

		// Build the event synchronously, then record with a background context
		evt := al.buildAuditEvent(c, statusCode, requestBody, startTime)
		if evt != nil {
			go func(e domain.Event) {
				// Use background context to avoid cancellation when request completes
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = al.auditStore.Record(ctx, e)
			}(*evt)
		}
	}
}

// buildAuditEvent creates an audit event from the request context.
func (al *AuditLogger) buildAuditEvent(c *gin.Context, statusCode int, requestBody []byte, startTime time.Time) *domain.Event {
	// Determine event type based on endpoint and status
	eventType := al.determineEventType(c.Request.URL.Path, c.Request.Method, statusCode)
	if eventType == domain.EventType("") {
		// Not an auditable event
		return nil
	}

	// Get user ID if authenticated
	var userID *uuid.UUID
	if ctx, ok := akauth.From(c); ok && ctx.IsAuthenticated() {
		userID = &ctx.UserID
	}
	// Fallback for refresh endpoint when context is missing
	if userID == nil && c.Request.URL.Path == "/auth/refresh" && al.refreshStore != nil {
		if cookie, err := httpkit.GetRefreshCookie(c.Request); err == nil && cookie != "" {
			if raw, err := base64.RawURLEncoding.DecodeString(cookie); err == nil && len(raw) == 32 {
				hash := crypto.HashSHA256(raw)
				if tok, err := al.refreshStore.GetByHash(c.Request.Context(), hash); err == nil && tok != nil {
					uid := tok.UserID
					userID = &uid
				}
			}
		}
	}

	// Get client IP
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = c.Request.RemoteAddr
	}

	// Build metadata (excluding sensitive data)
	metadata := map[string]interface{}{
		"method":      c.Request.Method,
		"path":        c.Request.URL.Path,
		"status":      statusCode,
		"ip":          clientIP,
		"user_agent":  c.Request.UserAgent(),
		"duration_ms": time.Since(startTime).Milliseconds(),
	}

	// Add sanitized request data for certain endpoints
	if len(requestBody) > 0 && shouldLogRequestBody(c.Request.URL.Path) {
		// Parse and sanitize request body (implementation depends on endpoint)
		sanitized := sanitizeRequestBody(c.Request.URL.Path, requestBody)
		if sanitized != nil {
			metadata["request"] = sanitized
		}
	}

	return &domain.Event{
		ID:        uuid.New(),
		Type:      eventType,
		UserID:    userID,
		ActorID:   userID, // For auth events, actor is the user
		IPAddress: clientIP,
		UserAgent: c.Request.UserAgent(),
		Metadata:  metadata,
		CreatedAt: startTime,
	}
}

// determineEventType maps endpoints to audit event types.
func (al *AuditLogger) determineEventType(path string, method string, statusCode int) domain.EventType {
	// Normalize method to uppercase for consistency
	method = strings.ToUpper(method)

	// Check for rate limiting first (applies to any endpoint)
	if statusCode == http.StatusTooManyRequests {
		return domain.EventRateLimitExceeded
	}

	// Map auth endpoints to event types
	switch {
	// Login events
	case path == "/auth/login/begin" && method == "POST":
		return domain.EventLoginBegin
	case path == "/auth/login/complete" && method == "POST":
		if statusCode < 400 {
			return domain.EventLoginSuccess
		}
		return domain.EventLoginFailed

	// Logout events
	case path == "/auth/logout" && method == "POST":
		return domain.EventLogout
	case path == "/auth/logout-all" && method == "POST":
		return domain.EventLogoutAll
	case path == "/auth/admin/users/force-logout" && method == "POST":
		return domain.EventAdminForceLogout

	// Registration/Onboarding events
	case path == "/auth/onboard/begin" && method == "POST":
		return domain.EventRegistrationBegin
	case path == "/auth/onboard/complete" && method == "POST":
		if statusCode < 400 {
			return domain.EventRegistrationSuccess
		}
		return domain.EventRegistrationFailed

	// Token refresh events
	case path == "/auth/refresh" && method == "POST":
		if statusCode < 400 {
			return domain.EventTokenRefresh
		} else if statusCode == http.StatusForbidden {
			// CSRF violation on refresh endpoint
			return domain.EventCSRFViolation
		}
		return domain.EventTokenRefreshFailed

	// Step-up authentication events
	case path == "/auth/stepup/begin" && method == "POST":
		return domain.EventStepUpBegin
	case path == "/auth/stepup/complete" && method == "POST":
		if statusCode < 400 {
			return domain.EventStepUpSuccess
		}
		return domain.EventStepUpFailed

	// Recovery events
	case path == "/auth/recovery/init" && method == "POST":
		return domain.EventRecoveryInitiated
	case path == "/auth/recovery/verify" && method == "POST":
		if statusCode < 400 {
			return domain.EventRecoveryCodeUsed
		}
		return domain.EventRecoveryFailed
	case path == "/auth/recovery/register/begin" && method == "POST":
		return domain.EventType("") // Part of recovery flow, not separately audited
	case path == "/auth/recovery/register/complete" && method == "POST":
		if statusCode < 400 {
			return domain.EventRecoveryCompleted
		}
		return domain.EventRecoveryFailed
	case path == "/auth/recovery/codes/generate" && method == "POST":
		if statusCode < 400 {
			return domain.EventRecoveryCodeGenerated
		}
		return domain.EventType("")

	// Credential management events
	case path == "/auth/credentials/add/begin" && method == "POST":
		return domain.EventType("") // Begin events not audited
	case path == "/auth/credentials/add/complete" && method == "POST":
		if statusCode < 400 {
			return domain.EventCredentialAdded
		}
		return domain.EventType("")
	case path == "/auth/credentials/remove" && method == "POST":
		if statusCode < 400 {
			return domain.EventCredentialRemoved
		}
		return domain.EventType("")
	case path == "/auth/credentials/revoke" && method == "POST":
		if statusCode < 400 {
			return domain.EventCredentialRevoked
		}
		return domain.EventType("")

	// Invite management events
	case path == "/auth/invites" && method == "POST":
		if statusCode < 400 {
			return domain.EventInviteCreated
		}
		return domain.EventType("")
	case strings.HasPrefix(path, "/auth/invites/") && method == "DELETE":
		if statusCode < 400 {
			return domain.EventInviteRevoked
		}
		return domain.EventType("")

	// Admin actions
	case strings.HasPrefix(path, "/auth/admin/") && method == "POST":
		if statusCode < 400 {
			return domain.EventAdminAction
		}
		return domain.EventType("")
	case strings.HasPrefix(path, "/auth/admin/users/") && strings.Contains(path, "/roles") && method == "PUT":
		if statusCode < 400 {
			return domain.EventUserRolesUpdated
		}
		return domain.EventType("")

	// General security events
	case statusCode == http.StatusForbidden:
		// Check if it's a CSRF error or general access denied
		if strings.HasPrefix(path, "/auth/refresh") {
			return domain.EventCSRFViolation
		}
		return domain.EventAccessDenied
	case statusCode == http.StatusUnauthorized:
		// Check the specific reason for 401
		if strings.Contains(path, "/auth") {
			// Could be expired JWT or invalid JWT
			return domain.EventInvalidJWT
		}
		return domain.EventType("")

	default:
		return domain.EventType("")
	}
}

// isAuthEndpoint checks if a path is an authentication endpoint.
func isAuthEndpoint(path string) bool {
	return strings.HasPrefix(path, "/auth")
}

// shouldLogRequestBody determines if request body should be logged for an endpoint.
func shouldLogRequestBody(path string) bool {
	// Only log request bodies for certain endpoints, never for sensitive ones
	switch path {
	case "/auth/invites", "/auth/credentials/remove":
		return true
	default:
		return false
	}
}

// sanitizeRequestBody removes sensitive data from request bodies.
// Deny-by-default: only returns data for explicitly vetted endpoints.
func sanitizeRequestBody(path string, body []byte) map[string]interface{} {
	// Never log WebAuthn responses (contain signatures)
	if bytes.Contains(body, []byte("clientDataJSON")) {
		return nil
	}

	// Never log anything with passwords, tokens, or codes
	if bytes.Contains(body, []byte("password")) ||
		bytes.Contains(body, []byte("token")) ||
		bytes.Contains(body, []byte("code")) ||
		bytes.Contains(body, []byte("secret")) {
		return nil
	}

	// Vetted endpoints with safe fields only
	// Currently no endpoints are vetted for body logging
	// Add specific parsing logic here only after security review

	// Default: return nil (deny-by-default)
	return nil
}

// responseWriter wraps gin.ResponseWriter to capture the status code.
type responseWriter struct {
	gin.ResponseWriter
	statusCode int
	written    bool
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	return w.ResponseWriter.Write(data)
}

// LogSecurityEvent logs a specific security event.
func (al *AuditLogger) LogSecurityEvent(c *gin.Context, eventType string, metadata map[string]interface{}) {
	// Get user ID if authenticated
	var userID *uuid.UUID
	if ctx, ok := akauth.From(c); ok && ctx.IsAuthenticated() {
		userID = &ctx.UserID
	}

	// Get client IP
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = c.Request.RemoteAddr
	}

	event := &domain.Event{
		ID:        uuid.New(),
		Type:      domain.EventType(eventType),
		UserID:    userID,
		ActorID:   userID,
		IPAddress: clientIP,
		UserAgent: c.Request.UserAgent(),
		Metadata:  metadata,
		CreatedAt: time.Now().UTC(),
	}

	// Log asynchronously with background context
	go func(e domain.Event) {
		// Use background context to avoid cancellation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = al.auditStore.Record(ctx, e)
	}(*event)
}

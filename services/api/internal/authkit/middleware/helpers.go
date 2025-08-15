package middleware

import (
    "time"

    "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
    basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
    "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rate"
    "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
    "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
    "github.com/gin-gonic/gin"
)

// Stack represents a collection of middleware components.
type Stack struct {
	auth     *Authenticator
	policies *PolicyEnforcer
	session  *SessionValidator
	rate     *RateLimiter
	audit    *AuditLogger
}

// NewStack creates a new middleware stack with all components.
func NewStack(
	tokenService service.TokenService,
	userStore store.UserStore,
	issuer string,
	registry *authkit.PolicyRegistry,
	kv store.KV,
	limiter rate.Limiter,
	auditStore store.AuditStore,
) *Stack {
	return &Stack{
		auth:     NewAuthenticator(tokenService, userStore, issuer),
		policies: NewPolicyEnforcer(registry),
		session:  NewSessionValidator(userStore, kv),
		rate:     NewRateLimiter(limiter),
		audit:    NewAuditLogger(auditStore),
	}
}

// Default returns the default middleware chain for most endpoints.
//
// This includes:
//   - Security headers
//   - CORS (if configured)
//   - Authentication
//   - Session validation
//   - Policy enforcement
//   - Audit logging
func (s *Stack) Default(corsConfig ...basehttpkit.CORSConfig) []gin.HandlerFunc {
	handlers := []gin.HandlerFunc{
		SecurityHeaders(),
	}

	// Add CORS if configured
    if len(corsConfig) > 0 {
        handlers = append(handlers, CORS(corsConfig[0]))
    }

	handlers = append(handlers,
		s.auth.Authenticate(),
		s.session.ValidateSession(),
		s.policies.EnforcePolicies(),
		s.audit.LogAuthEvents(),
	)

	return handlers
}

// Public returns middleware for public endpoints.
//
// This includes:
//   - Security headers
//   - CORS (if configured)
//   - Optional authentication
//   - Rate limiting
func (s *Stack) Public(rateLimit int, window time.Duration, corsConfig ...basehttpkit.CORSConfig) []gin.HandlerFunc {
	handlers := []gin.HandlerFunc{
		SecurityHeaders(),
	}

	// Add CORS if configured
    if len(corsConfig) > 0 {
        handlers = append(handlers, CORS(corsConfig[0]))
    }

	handlers = append(handlers,
		s.rate.LimitByIP(rateLimit, window),
		s.auth.OptionalAuth(),
	)

	return handlers
}

// Authenticated returns middleware for authenticated-only endpoints.
//
// This includes:
//   - Security headers
//   - Authentication (required)
//   - Session validation
//   - Rate limiting
//   - Audit logging
func (s *Stack) Authenticated(rateLimit int, window time.Duration) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		SecurityHeaders(),
		s.auth.Authenticate(),
		RequireAuth(),
		s.session.ValidateSession(),
		s.rate.LimitByUser(rateLimit, window),
		s.audit.LogAuthEvents(),
	}
}

// Admin returns middleware for admin-only endpoints.
//
// This includes:
//   - Security headers
//   - Authentication (required)
//   - Admin role requirement
//   - Session validation
//   - Step-up authentication
//   - Audit logging
func (s *Stack) Admin() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		SecurityHeaders(),
		s.auth.Authenticate(),
		RequireAuth(),
		RequireAdmin(),
		s.session.ValidateSession(),
		RequireStepUp(),
		s.audit.LogAuthEvents(),
	}
}

// AuthEndpoint returns middleware for authentication endpoints.
//
// This includes:
//   - Security headers
//   - Content-Type validation
//   - CSRF protection
//   - Rate limiting (endpoint-specific)
//   - Audit logging
func (s *Stack) AuthEndpoint(endpoint string) []gin.HandlerFunc {
	limits := DefaultAuthLimits()
    csrf := basehttpkit.NewHeaderCSRF()

	return []gin.HandlerFunc{
		SecurityHeaders(),
		ContentTypeJSON(),
		RequireCSRF(csrf),
		s.rate.ApplyAuthLimits(endpoint, limits),
		s.audit.LogAuthEvents(),
	}
}

// StepUpRequired returns middleware that requires recent step-up authentication.
//
// This includes:
//   - Security headers
//   - Authentication (required)
//   - Session validation
//   - Step-up requirement
//   - Audit logging
func (s *Stack) StepUpRequired() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		SecurityHeaders(),
		s.auth.Authenticate(),
		RequireAuth(),
		s.session.ValidateSession(),
		RequireStepUp(),
		s.audit.LogAuthEvents(),
	}
}

// Custom allows building a custom middleware chain.
type CustomChain struct {
	handlers []gin.HandlerFunc
	stack    *Stack
}

// NewCustomChain starts building a custom middleware chain.
func (s *Stack) Custom() *CustomChain {
	return &CustomChain{
		handlers: []gin.HandlerFunc{},
		stack:    s,
	}
}

// Add adds a middleware to the chain.
func (c *CustomChain) Add(handler gin.HandlerFunc) *CustomChain {
	c.handlers = append(c.handlers, handler)
	return c
}

// SecurityHeaders adds security headers.
func (c *CustomChain) SecurityHeaders() *CustomChain {
	return c.Add(SecurityHeaders())
}

// CORS adds CORS middleware.
func (c *CustomChain) CORS(config basehttpkit.CORSConfig) *CustomChain {
	return c.Add(CORS(config))
}

// Authenticate adds authentication middleware.
func (c *CustomChain) Authenticate() *CustomChain {
	return c.Add(c.stack.auth.Authenticate())
}

// RequireAuth adds required authentication.
func (c *CustomChain) RequireAuth() *CustomChain {
	return c.Add(RequireAuth())
}

// RequireRoles adds role requirement.
func (c *CustomChain) RequireRoles(roles ...string) *CustomChain {
	return c.Add(RequireRoles(roles...))
}

// RequireStepUp adds step-up requirement.
func (c *CustomChain) RequireStepUp() *CustomChain {
	return c.Add(RequireStepUp())
}

// ValidateSession adds session validation.
func (c *CustomChain) ValidateSession() *CustomChain {
	return c.Add(c.stack.session.ValidateSession())
}

// RateLimitByIP adds IP-based rate limiting.
func (c *CustomChain) RateLimitByIP(requests int, window time.Duration) *CustomChain {
	return c.Add(c.stack.rate.LimitByIP(requests, window))
}

// RateLimitByUser adds user-based rate limiting.
func (c *CustomChain) RateLimitByUser(requests int, window time.Duration) *CustomChain {
	return c.Add(c.stack.rate.LimitByUser(requests, window))
}

// AuditLog adds audit logging.
func (c *CustomChain) AuditLog() *CustomChain {
	return c.Add(c.stack.audit.LogAuthEvents())
}

// ContentTypeJSON adds JSON content type validation.
func (c *CustomChain) ContentTypeJSON() *CustomChain {
	return c.Add(ContentTypeJSON())
}

// CSRF adds CSRF protection.
func (c *CustomChain) CSRF(csrf basehttpkit.CSRF) *CustomChain {
	return c.Add(RequireCSRF(csrf))
}

// Build returns the final middleware chain.
func (c *CustomChain) Build() []gin.HandlerFunc {
	return c.handlers
}

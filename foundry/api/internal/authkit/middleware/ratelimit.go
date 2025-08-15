package middleware

import (
    "fmt"
    "math"
    "net/http"
    "strconv"
    "time"

    "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/authkit"
    basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
    "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/rate"
    "github.com/gin-gonic/gin"
)

// RateLimiter provides rate limiting middleware.
type RateLimiter struct {
	limiter rate.Limiter
}

// NewRateLimiter creates a new rate limiting middleware.
func NewRateLimiter(limiter rate.Limiter) *RateLimiter {
	return &RateLimiter{
		limiter: limiter,
	}
}

// LimitByIP creates middleware that rate limits by client IP address.
//
// Note: This relies on c.ClientIP() which depends on Gin's trusted proxy settings.
// Ensure the application configures SetTrustedProxies correctly to prevent
// IP spoofing via X-Forwarded-For headers.
func (rl *RateLimiter) LimitByIP(requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = c.Request.RemoteAddr
		}

		key := fmt.Sprintf("ip:%s:%s:%s", clientIP, c.Request.Method, c.Request.URL.Path)
		
		allowed, remaining, reset, err := rl.limiter.Allow(c.Request.Context(), rate.Key(key), requests, window)
		if err != nil {
			// Error checking rate limit, allow the request
			c.Next()
			return
		}
		// Add rate limit headers for transparency
		c.Header("X-RateLimit-Limit", strconv.Itoa(requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		
		if !allowed {
			// Add Retry-After header (rounded up)
			retryAfter := time.Until(reset)
			secs := int(math.Ceil(retryAfter.Seconds()))
			if secs < 0 {
				secs = 0
			}
			c.Header("Retry-After", strconv.Itoa(secs))
			
            basehttpkit.ErrorResponse(c.Writer, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests")
            c.Abort()
            return
        }

		c.Next()
	}
}

// LimitByUser creates middleware that rate limits by authenticated user.
func (rl *RateLimiter) LimitByUser(requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get auth context
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() {
			// No authenticated user, skip rate limiting
			c.Next()
			return
		}

		key := fmt.Sprintf("user:%s:%s:%s", ctx.UserID.String(), c.Request.Method, c.Request.URL.Path)
		
		allowed, remaining, reset, err := rl.limiter.Allow(c.Request.Context(), rate.Key(key), requests, window)
		if err != nil {
			// Error checking rate limit, allow the request
			c.Next()
			return
		}
		// Add rate limit headers for transparency
		c.Header("X-RateLimit-Limit", strconv.Itoa(requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		
		if !allowed {
			// Add Retry-After header (rounded up)
			retryAfter := time.Until(reset)
			secs := int(math.Ceil(retryAfter.Seconds()))
			if secs < 0 {
				secs = 0
			}
			c.Header("Retry-After", strconv.Itoa(secs))
			
            basehttpkit.ErrorResponse(c.Writer, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests")
            c.Abort()
            return
        }

		c.Next()
	}
}

// LimitByKey creates middleware that rate limits by a custom key.
func (rl *RateLimiter) LimitByKey(keyFunc func(*gin.Context) string, requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)
		if key == "" {
			// No key, skip rate limiting
			c.Next()
			return
		}

		allowed, remaining, reset, err := rl.limiter.Allow(c.Request.Context(), rate.Key(key), requests, window)
		if err != nil {
			// Error checking rate limit, allow the request
			c.Next()
			return
		}
		// Add rate limit headers for transparency
		c.Header("X-RateLimit-Limit", strconv.Itoa(requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		
		if !allowed {
			// Add Retry-After header (rounded up)
			retryAfter := time.Until(reset)
			secs := int(math.Ceil(retryAfter.Seconds()))
			if secs < 0 {
				secs = 0
			}
			c.Header("Retry-After", strconv.Itoa(secs))
			
            basehttpkit.ErrorResponse(c.Writer, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests")
            c.Abort()
            return
        }

		c.Next()
	}
}

// AuthEndpointLimits provides standard rate limits for authentication endpoints.
type AuthEndpointLimits struct {
	Login           *RateLimit
	InviteUse       *RateLimit
	RecoveryInit    *RateLimit
	RecoveryVerify  *RateLimit
	Refresh         *RateLimit
	CredentialsAdd  *RateLimit
}

// RateLimit defines a rate limit configuration.
type RateLimit struct {
	Requests int
	Window   time.Duration
}

// DefaultAuthLimits returns the standard rate limits from the spec.
func DefaultAuthLimits() AuthEndpointLimits {
	return AuthEndpointLimits{
		Login: &RateLimit{
			Requests: 5,
			Window:   15 * time.Minute,
		},
		InviteUse: &RateLimit{
			Requests: 5,
			Window:   0, // Lifetime limit, handled differently
		},
		RecoveryInit: &RateLimit{
			Requests: 3,
			Window:   time.Hour,
		},
		RecoveryVerify: &RateLimit{
			Requests: 5,
			Window:   time.Hour,
		},
		Refresh: &RateLimit{
			Requests: 100,
			Window:   time.Hour,
		},
		CredentialsAdd: &RateLimit{
			Requests: 10,
			Window:   time.Hour,
		},
	}
}

// ApplyAuthLimits creates middleware that applies standard auth endpoint rate limits.
func (rl *RateLimiter) ApplyAuthLimits(endpoint string, limits AuthEndpointLimits) gin.HandlerFunc {
	var limit *RateLimit

	switch endpoint {
	case "login":
		limit = limits.Login
	case "invite_use":
		// Invite use has lifetime limits, not time-window limits
		// This is enforced in the invite service itself
		return func(c *gin.Context) { c.Next() }
	case "recovery_init":
		limit = limits.RecoveryInit
	case "recovery_verify":
		limit = limits.RecoveryVerify
	case "refresh":
		limit = limits.Refresh
	case "credentials_add":
		limit = limits.CredentialsAdd
	default:
		// No limit defined for this endpoint
		return func(c *gin.Context) { c.Next() }
	}

	if limit == nil || limit.Requests == 0 {
		// No limit configured
		return func(c *gin.Context) { c.Next() }
	}

	// Use IP-based limiting for unauthenticated endpoints
	return rl.LimitByIP(limit.Requests, limit.Window)
}

// GlobalRateLimit applies a truly global rate limit across all users and routes.
func (rl *RateLimiter) GlobalRateLimit(requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use a single global key for true global limiting
		// Optionally include method: key := fmt.Sprintf("global:%s", c.Request.Method)
		key := "global"
		
		allowed, remaining, reset, err := rl.limiter.Allow(c.Request.Context(), rate.Key(key), requests, window)
		if err != nil {
			// Error checking rate limit, allow the request
			c.Next()
			return
		}
		// Add rate limit headers for transparency
		c.Header("X-RateLimit-Limit", strconv.Itoa(requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		
		if !allowed {
			// Add Retry-After header (rounded up)
			retryAfter := time.Until(reset)
			secs := int(math.Ceil(retryAfter.Seconds()))
			if secs < 0 {
				secs = 0
			}
			c.Header("Retry-After", strconv.Itoa(secs))
			
                basehttpkit.ErrorResponse(c.Writer, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}

// PerRouteGlobalRateLimit applies a global rate limit per route (method+path).
//
// This creates separate rate limit buckets for each unique route,
// but the limit applies globally to all users accessing that route.
func (rl *RateLimiter) PerRouteGlobalRateLimit(requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("global:%s:%s", c.Request.Method, c.Request.URL.Path)
		
		allowed, remaining, reset, err := rl.limiter.Allow(c.Request.Context(), rate.Key(key), requests, window)
		if err != nil {
			// Error checking rate limit, allow the request
			c.Next()
			return
		}
		// Add rate limit headers for transparency
		c.Header("X-RateLimit-Limit", strconv.Itoa(requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		
		if !allowed {
			// Add Retry-After header (rounded up)
			retryAfter := time.Until(reset)
			secs := int(math.Ceil(retryAfter.Seconds()))
			if secs < 0 {
				secs = 0
			}
			c.Header("Retry-After", strconv.Itoa(secs))
			
                basehttpkit.ErrorResponse(c.Writer, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}

package middleware

import (
    "fmt"
    "net/http"
    "strconv"
    "time"

    basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
    "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/rate"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

// RateLimiterV2 provides rate limiting middleware using the rate package.
type RateLimiterV2 struct {
	limiter  rate.Limiter
	keyFunc  func(c *gin.Context) rate.Key
	duration time.Duration
}

// NewRateLimiterV2 creates a new rate limiter middleware.
func NewRateLimiterV2(limiter rate.Limiter, duration time.Duration) *RateLimiterV2 {
	return &RateLimiterV2{
		limiter:  limiter,
		duration: duration,
		keyFunc:  defaultKeyFunc,
	}
}

// defaultKeyFunc uses IP address as the rate limit key.
func defaultKeyFunc(c *gin.Context) rate.Key {
	return rate.Key("ip:" + c.ClientIP())
}

// SetKeyFunc sets a custom key function.
func (rl *RateLimiterV2) SetKeyFunc(fn func(c *gin.Context) rate.Key) {
	rl.keyFunc = fn
}

// Middleware returns a gin middleware that enforces rate limits.
func (rl *RateLimiterV2) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := rl.keyFunc(c)
		
		allowed, remaining, resetAt, err := rl.limiter.Allow(c.Request.Context(), key, 1, rl.duration)
		if err != nil {
			// Log error but don't fail the request
			// In production, you'd want to monitor this
			c.Next()
			return
		}
		
		// Always set rate limit headers
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		
		if !allowed {
			retryAfter := int(time.Until(resetAt).Seconds())
			if retryAfter > 0 {
				c.Header("Retry-After", strconv.Itoa(retryAfter))
			}
			
            basehttpkit.ErrorResponse(
                c.Writer,
                http.StatusTooManyRequests,
                "rate_limit_exceeded",
                "Too many requests. Please try again later.",
            )
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// LoginRateLimiter creates a rate limiter for login attempts.
func LoginRateLimiter(limiter rate.Limiter) gin.HandlerFunc {
	rl := NewRateLimiterV2(limiter, 15*time.Minute)
	rl.SetKeyFunc(func(c *gin.Context) rate.Key {
		// Rate limit by email if available
		var req struct {
			Email string `json:"email"`
		}
		if c.ShouldBindJSON(&req) == nil && req.Email != "" {
			return rate.KeyLogin(req.Email)
		}
		// Fall back to IP
		return rate.Key("login:ip:" + c.ClientIP())
	})
	return rl.Middleware()
}

// InviteRateLimiter creates a rate limiter for invite attempts.
func InviteRateLimiter(limiter rate.Limiter) gin.HandlerFunc {
	rl := NewRateLimiterV2(limiter, time.Hour)
	rl.SetKeyFunc(func(c *gin.Context) rate.Key {
		// Rate limit by invite ID if available
		inviteID := c.Param("id")
		if inviteID != "" {
			if id, err := uuid.Parse(inviteID); err == nil {
				return rate.KeyInvite(id)
			}
		}
		// Fall back to IP
		return rate.Key("invite:ip:" + c.ClientIP())
	})
	return rl.Middleware()
}

// RecoveryInitRateLimiter creates a rate limiter for recovery initiation.
func RecoveryInitRateLimiter(limiter rate.Limiter) gin.HandlerFunc {
	rl := NewRateLimiterV2(limiter, time.Hour)
	rl.SetKeyFunc(func(c *gin.Context) rate.Key {
		// Rate limit by email if available
		var req struct {
			Email string `json:"email"`
		}
		if c.ShouldBindJSON(&req) == nil && req.Email != "" {
			return rate.Key("recovery:init:" + req.Email)
		}
		// Fall back to IP
		return rate.Key("recovery:init:ip:" + c.ClientIP())
	})
	return rl.Middleware()
}

// RecoveryVerifyRateLimiter creates a rate limiter for recovery verification.
func RecoveryVerifyRateLimiter(limiter rate.Limiter) gin.HandlerFunc {
	rl := NewRateLimiterV2(limiter, time.Hour)
	rl.SetKeyFunc(func(c *gin.Context) rate.Key {
		// Rate limit by flow ID if available
		var req struct {
			FlowID string `json:"flow_id"`
		}
		if c.ShouldBindJSON(&req) == nil && req.FlowID != "" {
			return rate.Key("recovery:verify:" + req.FlowID)
		}
		// Fall back to IP
		return rate.Key("recovery:verify:ip:" + c.ClientIP())
	})
	return rl.Middleware()
}

// RefreshRateLimiter creates a rate limiter for token refresh.
func RefreshRateLimiter(limiter rate.Limiter) gin.HandlerFunc {
	rl := NewRateLimiterV2(limiter, time.Hour)
	rl.SetKeyFunc(func(c *gin.Context) rate.Key {
		// Rate limit by user ID if authenticated
		if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(uuid.UUID); ok {
				return rate.KeyRefresh(id)
			}
			if idStr, ok := userID.(string); ok {
				if id, err := uuid.Parse(idStr); err == nil {
					return rate.KeyRefresh(id)
				}
			}
		}
		// Fall back to IP
		return rate.Key("refresh:ip:" + c.ClientIP())
	})
	return rl.Middleware()
}

// CredentialAddRateLimiter creates a rate limiter for adding credentials.
func CredentialAddRateLimiter(limiter rate.Limiter) gin.HandlerFunc {
	rl := NewRateLimiterV2(limiter, time.Hour)
	rl.SetKeyFunc(func(c *gin.Context) rate.Key {
		// Rate limit by user ID if authenticated
		if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(uuid.UUID); ok {
				return rate.KeyCredentialAdd(id)
			}
			if idStr, ok := userID.(string); ok {
				if id, err := uuid.Parse(idStr); err == nil {
					return rate.KeyCredentialAdd(id)
				}
			}
		}
		// Fall back to IP
		return rate.Key("credential_add:ip:" + c.ClientIP())
	})
	return rl.Middleware()
}

// GlobalRateLimiter creates a general purpose rate limiter.
func GlobalRateLimiter(limiter rate.Limiter, requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a key based on IP and path
		key := rate.Key(fmt.Sprintf("global:%s:%s:%s", c.ClientIP(), c.Request.Method, c.Request.URL.Path))
		
		allowed, remaining, resetAt, err := limiter.Allow(c.Request.Context(), key, 1, window)
		if err != nil {
			c.Next()
			return
		}
		
		// Set headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
		
		if !allowed {
			retryAfter := int(time.Until(resetAt).Seconds())
			if retryAfter > 0 {
				c.Header("Retry-After", strconv.Itoa(retryAfter))
			}
			
            basehttpkit.ErrorResponse(
                c.Writer,
                http.StatusTooManyRequests,
                "rate_limit_exceeded",
                "Too many requests. Please try again later.",
            )
			c.Abort()
			return
		}
		
		c.Next()
	}
}

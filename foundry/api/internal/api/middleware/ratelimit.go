package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type tokenBucket struct {
	capacity int
	tokens   int
	refill   time.Duration
	last     time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	capacity int
	refill   time.Duration
}

func NewRateLimiter(capacity int, refill time.Duration) *RateLimiter {
	return &RateLimiter{buckets: make(map[string]*tokenBucket), capacity: capacity, refill: refill}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[key]
	now := time.Now()
	if !ok {
		rl.buckets[key] = &tokenBucket{capacity: rl.capacity, tokens: rl.capacity - 1, refill: rl.refill, last: now}
		return true
	}
	// Refill
	elapsed := now.Sub(b.last)
	if elapsed >= b.refill {
		add := int(elapsed / b.refill)
		b.tokens = min(b.capacity, b.tokens+add)
		b.last = now
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RateLimit returns a Gin middleware that limits requests per client IP.
func RateLimit(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			ip = c.ClientIP()
		}
		if !rl.Allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

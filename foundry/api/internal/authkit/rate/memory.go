package rate

import (
	"context"
	"strings"
	"sync"
	"time"
)

// bucket represents a rate limit bucket.
type bucket struct {
	count     int
	resetTime time.Time
}

// MemoryLimiter implements an in-memory rate limiter.
type MemoryLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	
	// For cleanup
	stopCleanup chan struct{}
	cleanupDone chan struct{}
}

// NewMemoryLimiter creates a new memory-based rate limiter.
func NewMemoryLimiter() *MemoryLimiter {
	ml := &MemoryLimiter{
		buckets:     make(map[string]*bucket),
		stopCleanup: make(chan struct{}),
		cleanupDone: make(chan struct{}),
	}
	
	// Start cleanup goroutine
	go ml.cleanup()
	
	return ml
}

// Allow checks if n requests are allowed for the given key within the specified duration.
func (ml *MemoryLimiter) Allow(ctx context.Context, key Key, n int, per time.Duration) (bool, int, time.Time, error) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	
	now := time.Now().UTC()
	bucketKey := string(key)
	
	// Get or create bucket
	b, exists := ml.buckets[bucketKey]
	if !exists || now.After(b.resetTime) {
		// New window or expired bucket
		limit := ml.getLimit(key, per)
		b = &bucket{
			count:     0,
			resetTime: now.Add(per),
		}
		ml.buckets[bucketKey] = b
		
		// Check if request fits in limit
		if n <= limit {
			b.count = n
			return true, limit - n, b.resetTime, nil
		}
		return false, 0, b.resetTime, nil
	}
	
	// Existing window
	limit := ml.getLimit(key, per)
	if b.count + n <= limit {
		b.count += n
		return true, limit - b.count, b.resetTime, nil
	}
	
	return false, 0, b.resetTime, nil
}

// getLimit returns the rate limit for a given key and duration.
func (ml *MemoryLimiter) getLimit(key Key, per time.Duration) int {
	keyStr := string(key)
	
	// Determine limits based on key prefix and duration
	switch {
	case strings.HasPrefix(keyStr, "login:"):
		// Login: 5 attempts per 15 minutes
		if per == 15*time.Minute {
			return 5
		}
		// Scale proportionally for other durations
		return int(5 * (per.Minutes() / 15))
		
	case strings.HasPrefix(keyStr, "invite:"):
		// Invite: 5 attempts per lifetime (use 10 per hour as backstop)
		return 10
		
	case strings.HasPrefix(keyStr, "recovery:"):
		// Recovery: 3 attempts per hour for init, 5 for verify
		if strings.Contains(keyStr, ":init") {
			if per == time.Hour {
				return 3
			}
			return int(3 * (per.Hours()))
		}
		// Recovery verify
		if per == time.Hour {
			return 5
		}
		return int(5 * (per.Hours()))
		
	case strings.HasPrefix(keyStr, "refresh:"):
		// Refresh: 100 attempts per hour
		if per == time.Hour {
			return 100
		}
		return int(100 * (per.Hours()))
		
	case strings.HasPrefix(keyStr, "credential_add:"):
		// Credentials add: 10 attempts per hour
		if per == time.Hour {
			return 10
		}
		return int(10 * (per.Hours()))
		
	default:
		// Default: 60 requests per minute as a reasonable default
		return int(60 * (per.Minutes()))
	}
}

// Close stops the cleanup goroutine.
func (ml *MemoryLimiter) Close() {
	close(ml.stopCleanup)
	<-ml.cleanupDone
}

// cleanup periodically removes old buckets.
func (ml *MemoryLimiter) cleanup() {
	defer close(ml.cleanupDone)
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ml.stopCleanup:
			return
		case <-ticker.C:
			ml.cleanupBuckets()
		}
	}
}

// cleanupBuckets removes expired buckets.
func (ml *MemoryLimiter) cleanupBuckets() {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	
	now := time.Now().UTC()
	for key, b := range ml.buckets {
		// Remove buckets that have been expired for more than an hour
		if now.Sub(b.resetTime) > time.Hour {
			delete(ml.buckets, key)
		}
	}
}
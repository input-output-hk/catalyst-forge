package rate

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DeviceLinkLimiter provides rate limiting for device-link operations.
type DeviceLinkLimiter interface {
	// AllowBegin checks if a new device-link flow can be started (per IP).
	AllowBegin(ctx context.Context, ip string) (bool, error)
	
	// AllowExchange checks if a device code exchange attempt is allowed.
	AllowExchange(ctx context.Context, deviceCode string) (bool, error)
	
	// RecordSlowDown records that a client should slow down polling.
	RecordSlowDown(ctx context.Context, deviceCode string) error
	
	// ResetExchange resets the rate limit for a device code (on success).
	ResetExchange(ctx context.Context, deviceCode string) error
}

// deviceLinkLimiter implements DeviceLinkLimiter with in-memory storage.
type deviceLinkLimiter struct {
	mu              sync.RWMutex
	beginAttempts   map[string]*rateBucket   // IP -> attempts
	exchangeAttempts map[string]*rateBucket  // deviceCode -> attempts
	slowDownUntil   map[string]time.Time     // deviceCode -> slow down until
	
	beginLimit      int           // Max begins per window per IP
	beginWindow     time.Duration // Time window for begin attempts
	exchangeLimit   int           // Max exchanges per window per device code
	exchangeWindow  time.Duration // Time window for exchange attempts
	minInterval     time.Duration // Minimum polling interval
}

// rateBucket tracks attempts within a time window.
type rateBucket struct {
	count      int
	windowStart time.Time
}

// NewDeviceLinkLimiter creates a new device-link rate limiter.
func NewDeviceLinkLimiter() DeviceLinkLimiter {
	return &deviceLinkLimiter{
		beginAttempts:    make(map[string]*rateBucket),
		exchangeAttempts: make(map[string]*rateBucket),
		slowDownUntil:    make(map[string]time.Time),
		beginLimit:       10,              // 10 begins per IP per hour
		beginWindow:      time.Hour,
		exchangeLimit:    120,             // 120 exchanges per 10 minutes (every 5s)
		exchangeWindow:   10 * time.Minute,
		minInterval:      5 * time.Second, // 5 second minimum between polls
	}
}

// AllowBegin checks if a new device-link flow can be started.
func (l *deviceLinkLimiter) AllowBegin(ctx context.Context, ip string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	now := time.Now()
	bucket, exists := l.beginAttempts[ip]
	
	// Create new bucket if needed or reset if window expired
	if !exists || now.Sub(bucket.windowStart) > l.beginWindow {
		l.beginAttempts[ip] = &rateBucket{
			count:       1,
			windowStart: now,
		}
		return true, nil
	}
	
	// Check if within limit
	if bucket.count >= l.beginLimit {
		return false, nil
	}
	
	bucket.count++
	return true, nil
}

// AllowExchange checks if a device code exchange attempt is allowed.
func (l *deviceLinkLimiter) AllowExchange(ctx context.Context, deviceCode string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	now := time.Now()
	
	// Check if in slow-down period
	if slowDownUntil, exists := l.slowDownUntil[deviceCode]; exists {
		if now.Before(slowDownUntil) {
			return false, fmt.Errorf("slow_down")
		}
		delete(l.slowDownUntil, deviceCode)
	}
	
	bucket, exists := l.exchangeAttempts[deviceCode]
	
	// Create new bucket if needed or reset if window expired
	if !exists || now.Sub(bucket.windowStart) > l.exchangeWindow {
		l.exchangeAttempts[deviceCode] = &rateBucket{
			count:       1,
			windowStart: now,
		}
		return true, nil
	}
	
	// Check if within limit
	if bucket.count >= l.exchangeLimit {
		return false, nil
	}
	
	bucket.count++
	return true, nil
}

// RecordSlowDown records that a client should slow down polling.
func (l *deviceLinkLimiter) RecordSlowDown(ctx context.Context, deviceCode string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Tell client to slow down for 30 seconds
	l.slowDownUntil[deviceCode] = time.Now().Add(30 * time.Second)
	return nil
}

// ResetExchange resets the rate limit for a device code.
func (l *deviceLinkLimiter) ResetExchange(ctx context.Context, deviceCode string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	delete(l.exchangeAttempts, deviceCode)
	delete(l.slowDownUntil, deviceCode)
	return nil
}

// Cleanup removes old entries to prevent memory growth.
func (l *deviceLinkLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	now := time.Now()
	
	// Clean up old begin attempts
	for ip, bucket := range l.beginAttempts {
		if now.Sub(bucket.windowStart) > l.beginWindow {
			delete(l.beginAttempts, ip)
		}
	}
	
	// Clean up old exchange attempts
	for code, bucket := range l.exchangeAttempts {
		if now.Sub(bucket.windowStart) > l.exchangeWindow {
			delete(l.exchangeAttempts, code)
		}
	}
	
	// Clean up expired slow-down entries
	for code, until := range l.slowDownUntil {
		if now.After(until) {
			delete(l.slowDownUntil, code)
		}
	}
}

// StartCleanupWorker starts a background worker to clean up old entries.
func StartDeviceLinkCleanupWorker(limiter *deviceLinkLimiter, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for range ticker.C {
			limiter.cleanup()
		}
	}()
}
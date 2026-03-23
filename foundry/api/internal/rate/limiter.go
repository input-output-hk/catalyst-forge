package rate

import (
	"context"
	"sync"
	"time"
)

// Limiter provides a simple per-key rate limit interface
type Limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// InMemoryLimiter is a simple in-memory limiter suitable for dev/tests
type InMemoryLimiter struct {
	mu   sync.Mutex
	data map[string][]time.Time
}

func NewInMemoryLimiter() *InMemoryLimiter {
	return &InMemoryLimiter{data: make(map[string][]time.Time)}
}

func (l *InMemoryLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now()
	cutoff := now.Add(-window)
	l.mu.Lock()
	defer l.mu.Unlock()
	timestamps := l.data[key]
	// drop old
	i := 0
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			timestamps[i] = ts
			i++
		}
	}
	timestamps = timestamps[:i]
	if len(timestamps) >= limit {
		l.data[key] = timestamps
		return false, nil
	}
	timestamps = append(timestamps, now)
	l.data[key] = timestamps
	return true, nil
}

package store

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a key is not found.
	ErrNotFound = errors.New("key not found")
)

// KV provides ephemeral key-value storage for step-up grants and other temporary data.
type KV interface {
	// Set stores a value with an optional TTL.
	//
	// A TTL of 0 means no expiration.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	
	// Get retrieves a value by key.
	//
	// Returns nil if the key doesn't exist or has expired.
	Get(ctx context.Context, key string) ([]byte, error)
	
	// Del deletes a key.
	Del(ctx context.Context, key string) error
}
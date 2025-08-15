package store

import (
	"context"
	"time"
)

// ChallengeStore handles temporary challenge storage for WebAuthn ceremonies.
type ChallengeStore interface {
	// Set stores a challenge with a TTL.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	
	// Get retrieves a challenge without deleting it.
	//
	// Returns error if the challenge doesn't exist or has expired.
	Get(ctx context.Context, key string) ([]byte, error)
	
	// Delete removes a challenge.
	Delete(ctx context.Context, key string) error
	
	// Take atomically retrieves and deletes a challenge.
	//
	// Returns nil if the challenge doesn't exist or has expired.
	Take(ctx context.Context, key string) ([]byte, error)
}
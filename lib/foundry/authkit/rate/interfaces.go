package rate

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Key represents a rate limit key.
type Key string

// Limiter provides rate limiting functionality.
type Limiter interface {
	// Allow checks if n requests are allowed for the given key within the specified duration.
	//
	// Returns whether the request is allowed, remaining requests, reset time, and any error.
	Allow(ctx context.Context, key Key, n int, per time.Duration) (ok bool, remaining int, reset time.Time, err error)
}

// KeyLogin creates a rate limit key for login attempts.
func KeyLogin(email string) Key {
	return Key("login:" + email)
}

// KeyInvite creates a rate limit key for invite redemption attempts.
func KeyInvite(id uuid.UUID) Key {
	return Key("invite:" + id.String())
}

// KeyRecovery creates a rate limit key for recovery attempts.
func KeyRecovery(email string) Key {
	return Key("recovery:" + email)
}

// KeyRefresh creates a rate limit key for token refresh attempts.
func KeyRefresh(tokenID uuid.UUID) Key {
	return Key("refresh:" + tokenID.String())
}

// KeyCredentialAdd creates a rate limit key for adding credentials.
func KeyCredentialAdd(userID uuid.UUID) Key {
	return Key("credential_add:" + userID.String())
}
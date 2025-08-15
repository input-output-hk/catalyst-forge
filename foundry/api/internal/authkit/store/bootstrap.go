package store

import "context"

// BootstrapStore records one-time bootstrap token usage for admin initialization.
type BootstrapStore interface {
	// IsTokenUsed returns true if the given SHA-256 hex token hash has been used.
	IsTokenUsed(ctx context.Context, tokenHashHex string) (bool, error)
	// MarkUsed records the token as used by the given email. Must enforce uniqueness
	// on token hash to prevent replay under race conditions.
	MarkUsed(ctx context.Context, tokenHashHex string, email string) error
}

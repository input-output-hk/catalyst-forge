package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RecoveryCodeStore handles recovery code persistence.
type RecoveryCodeStore interface {
	// ReplaceCodes replaces all recovery codes for a user with new ones.
	//
	// This operation should be atomic - either all codes are replaced or none are.
	ReplaceCodes(ctx context.Context, userID uuid.UUID, codeHashes [][]byte) error
	
	// Consume attempts to use a recovery code, marking it as used if found.
	//
	// Returns true if the code was valid and consumed, false otherwise.
	Consume(ctx context.Context, userID uuid.UUID, codeHash []byte, at time.Time) (bool, error)
	
	// List returns all recovery code hashes for a user.
	//
	// This is primarily for admin operations.
	List(ctx context.Context, userID uuid.UUID) ([][]byte, error)
	
	// CountUnused returns the number of unused recovery codes for a user.
	//
	// This is used to show users how many codes they have left.
	CountUnused(ctx context.Context, userID uuid.UUID) (int, error)
}
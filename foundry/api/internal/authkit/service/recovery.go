package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
	"github.com/google/uuid"
)

var (
	// ErrRecoveryCodeInvalid is returned when a recovery code is invalid or already used.
	ErrRecoveryCodeInvalid = errors.New("recovery code is invalid or already used")
	// ErrNoRecoveryCodes is returned when no recovery codes are available.
	ErrNoRecoveryCodes = errors.New("no recovery codes available")
)

// RecoveryService handles account recovery operations.
type RecoveryService interface {
	// GenerateCodes creates a new set of recovery codes for a user.
	// Returns the plaintext codes (to show once to the user) and stores hashes.
	GenerateCodes(ctx context.Context, userID uuid.UUID, count int) ([]string, error)
	
	// ValidateCode checks if a recovery code is valid and marks it as used.
	// Returns an error if the code is invalid or already used.
	ValidateCode(ctx context.Context, userID uuid.UUID, code string) error
	
	// GetRemainingCount returns the number of unused recovery codes for a user.
	GetRemainingCount(ctx context.Context, userID uuid.UUID) (int, error)
	
	// RegenerateCodes replaces all existing codes with a new set.
	RegenerateCodes(ctx context.Context, userID uuid.UUID, count int) ([]string, error)
}

// recoveryService implements RecoveryService.
type recoveryService struct {
	store     store.RecoveryCodeStore
	rand      crypto.Rand
	codeLen   int
	codeCount int // Default number of codes to generate
}

// NewRecoveryService creates a new recovery service.
func NewRecoveryService(store store.RecoveryCodeStore, rand crypto.Rand) RecoveryService {
	return &recoveryService{
		store:     store,
		rand:      rand,
		codeLen:   16, // 16 bytes = 128 bits of entropy
		codeCount: 10, // Default to 10 recovery codes
	}
}

// GenerateCodes creates a new set of recovery codes for a user.
func (s *recoveryService) GenerateCodes(ctx context.Context, userID uuid.UUID, count int) ([]string, error) {
	if count <= 0 {
		count = s.codeCount // Use default if not specified
	}
	
	codes := make([]string, count)
	hashes := make([][]byte, count)
	
	for i := 0; i < count; i++ {
		// Generate random bytes
		codeBytes, err := s.rand.Bytes(s.codeLen)
		if err != nil {
			return nil, fmt.Errorf("failed to generate recovery code: %w", err)
		}
		
		// Encode as base64 for display
		code := base64.RawURLEncoding.EncodeToString(codeBytes)
		codes[i] = code
		
		// Store the hash
		hashes[i] = crypto.HashSHA256(codeBytes)
	}
	
	// Replace all codes atomically
	if err := s.store.ReplaceCodes(ctx, userID, hashes); err != nil {
		return nil, fmt.Errorf("failed to store recovery codes: %w", err)
	}
	
	return codes, nil
}

// ValidateCode checks if a recovery code is valid and marks it as used.
func (s *recoveryService) ValidateCode(ctx context.Context, userID uuid.UUID, code string) error {
	// Decode the code
	codeBytes, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil {
		return ErrRecoveryCodeInvalid
	}
	
	// Sanity check the code length
	if len(codeBytes) != s.codeLen {
		return ErrRecoveryCodeInvalid
	}
	
	// Hash the code
	hash := crypto.HashSHA256(codeBytes)
	
	// Try to consume the code
	consumed, err := s.store.Consume(ctx, userID, hash, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to consume recovery code: %w", err)
	}
	
	if !consumed {
		return ErrRecoveryCodeInvalid
	}
	
	return nil
}

// GetRemainingCount returns the number of unused recovery codes for a user.
func (s *recoveryService) GetRemainingCount(ctx context.Context, userID uuid.UUID) (int, error) {
	// Get the count of unused codes
	count, err := s.store.CountUnused(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count unused recovery codes: %w", err)
	}
	
	return count, nil
}

// RegenerateCodes replaces all existing codes with a new set.
func (s *recoveryService) RegenerateCodes(ctx context.Context, userID uuid.UUID, count int) ([]string, error) {
	// GenerateCodes already uses ReplaceCodes which is atomic
	return s.GenerateCodes(ctx, userID, count)
}
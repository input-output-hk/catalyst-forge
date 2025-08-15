package inmemory

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/google/uuid"
)

// RecoveryCodeStore is an in-memory implementation of store.RecoveryCodeStore.
type RecoveryCodeStore struct {
	mu    sync.RWMutex
	codes map[uuid.UUID][]*domain.RecoveryCode // UserID -> codes
}

// NewRecoveryCodeStore creates a new in-memory recovery code store.
func NewRecoveryCodeStore() *RecoveryCodeStore {
	return &RecoveryCodeStore{
		codes: make(map[uuid.UUID][]*domain.RecoveryCode),
	}
}

// ReplaceCodes replaces all recovery codes for a user with new ones.
//
// This operation is atomic - either all codes are replaced or none are.
func (s *RecoveryCodeStore) ReplaceCodes(ctx context.Context, userID uuid.UUID, codeHashes [][]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	newCodes := make([]*domain.RecoveryCode, len(codeHashes))
	
	for i, hash := range codeHashes {
		// Create a copy of the hash to avoid external modifications
		hashCopy := make([]byte, len(hash))
		copy(hashCopy, hash)
		
		newCodes[i] = &domain.RecoveryCode{
			UserID:    userID,
			Hash:      hashCopy,
			CreatedAt: now,
		}
	}

	s.codes[userID] = newCodes
	return nil
}

// Consume attempts to use a recovery code, marking it as used if found.
//
// Returns true if the code was valid and consumed, false otherwise.
func (s *RecoveryCodeStore) Consume(ctx context.Context, userID uuid.UUID, codeHash []byte, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userCodes, exists := s.codes[userID]
	if !exists {
		return false, nil
	}

	for _, code := range userCodes {
		if bytes.Equal(code.Hash, codeHash) {
			if code.UsedAt != nil {
				// Code already used
				return false, nil
			}
			// Mark as used
			code.UsedAt = &at
			return true, nil
		}
	}

	return false, nil
}

// List returns all recovery code hashes for a user.
//
// This is primarily for admin operations.
func (s *RecoveryCodeStore) List(ctx context.Context, userID uuid.UUID) ([][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userCodes, exists := s.codes[userID]
	if !exists {
		return [][]byte{}, nil
	}

	result := make([][]byte, 0, len(userCodes))
	for _, code := range userCodes {
		if code.UsedAt == nil {
			// Return a copy of the hash
			hashCopy := make([]byte, len(code.Hash))
			copy(hashCopy, code.Hash)
			result = append(result, hashCopy)
		}
	}

	return result, nil
}

// CountUnused returns the number of unused recovery codes for a user.
//
// This is used to show users how many codes they have left.
func (s *RecoveryCodeStore) CountUnused(ctx context.Context, userID uuid.UUID) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userCodes, exists := s.codes[userID]
	if !exists {
		return 0, nil
	}

	count := 0
	for _, code := range userCodes {
		if code.UsedAt == nil {
			count++
		}
	}

	return count, nil
}

// GetByUserID returns all recovery codes for a user (for testing).
//
// This method is not part of the store.RecoveryCodeStore interface.
func (s *RecoveryCodeStore) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.RecoveryCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userCodes, exists := s.codes[userID]
	if !exists {
		return []*domain.RecoveryCode{}, nil
	}

	// Return a copy of the codes
	result := make([]*domain.RecoveryCode, len(userCodes))
	copy(result, userCodes)
	return result, nil
}

// GetByHash returns a recovery code by its hash (for testing).
//
// This method is not part of the store.RecoveryCodeStore interface.
func (s *RecoveryCodeStore) GetByHash(ctx context.Context, userID uuid.UUID, codeHash []byte) (*domain.RecoveryCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userCodes, exists := s.codes[userID]
	if !exists {
		return nil, nil
	}

	for _, code := range userCodes {
		if bytes.Equal(code.Hash, codeHash) {
			return code, nil
		}
	}

	return nil, nil
}
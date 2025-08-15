package inmemory

import (
	"context"
	"errors"
	"sync"
	"time"
)

// challengeEntry stores a challenge value with its expiration time.
type challengeEntry struct {
	value     []byte
	expiresAt time.Time
}

// ChallengeStore is an in-memory implementation of store.ChallengeStore.
type ChallengeStore struct {
	mu         sync.RWMutex
	challenges map[string]*challengeEntry
}

// NewChallengeStore creates a new in-memory challenge store.
func NewChallengeStore() *ChallengeStore {
	return &ChallengeStore{
		challenges: make(map[string]*challengeEntry),
	}
}

// Set stores a challenge with a TTL.
func (s *ChallengeStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Make a copy of the value
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	s.challenges[key] = &challengeEntry{
		value:     valueCopy,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Get retrieves a challenge without deleting it.
//
// Returns error if the challenge doesn't exist or has expired.
func (s *ChallengeStore) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	entry, exists := s.challenges[key]
	if !exists {
		return nil, errors.New("challenge not found")
	}
	
	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return nil, errors.New("challenge expired")
	}
	
	// Return a copy of the value
	result := make([]byte, len(entry.value))
	copy(result, entry.value)
	
	return result, nil
}

// Delete removes a challenge.
func (s *ChallengeStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.challenges, key)
	return nil
}

// Take atomically retrieves and deletes a challenge.
//
// Returns nil if the challenge doesn't exist or has expired.
func (s *ChallengeStore) Take(ctx context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.challenges[key]
	if !exists {
		return nil, nil
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		delete(s.challenges, key)
		return nil, nil
	}

	// Delete and return value
	delete(s.challenges, key)
	return entry.value, nil
}

// cleanup removes expired challenges (for testing/maintenance).
//
// This method is not part of the store.ChallengeStore interface.
func (s *ChallengeStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, entry := range s.challenges {
		if now.After(entry.expiresAt) {
			delete(s.challenges, key)
		}
	}
}
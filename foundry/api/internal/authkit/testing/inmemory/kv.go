package inmemory

import (
	"context"
	"sync"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
)

// kvEntry stores a value with optional expiration.
type kvEntry struct {
	value     []byte
	expiresAt *time.Time
}

// KV is an in-memory implementation of store.KV.
type KV struct {
	mu      sync.RWMutex
	entries map[string]*kvEntry
}

// NewKV creates a new in-memory key-value store.
func NewKV() *KV {
	return &KV{
		entries: make(map[string]*kvEntry),
	}
}

// Set stores a value with an optional TTL.
//
// A TTL of 0 means no expiration.
func (s *KV) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Make a copy of the value
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	entry := &kvEntry{
		value: valueCopy,
	}

	if ttl > 0 {
		expiresAt := time.Now().Add(ttl)
		entry.expiresAt = &expiresAt
	}

	s.entries[key] = entry
	return nil
}

// Get retrieves a value by key.
//
// Returns store.ErrNotFound if the key doesn't exist or has expired.
func (s *KV) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.entries[key]
	if !exists {
		return nil, store.ErrNotFound
	}

	// Check if expired
	if entry.expiresAt != nil && time.Now().After(*entry.expiresAt) {
		return nil, store.ErrNotFound
	}

	// Return a copy of the value
	valueCopy := make([]byte, len(entry.value))
	copy(valueCopy, entry.value)

	return valueCopy, nil
}

// Del deletes a key.
func (s *KV) Del(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.entries, key)
	return nil
}

// Cleanup removes expired entries (for testing/maintenance).
//
// This method is not part of the store.KV interface.
func (s *KV) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, entry := range s.entries {
		if entry.expiresAt != nil && now.After(*entry.expiresAt) {
			delete(s.entries, key)
		}
	}
}
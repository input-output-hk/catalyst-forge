package memstore

import (
	"context"
	"sync"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
)

// ChallengeStore is an in-memory ChallengeStore with TTL semantics evaluated at access time.
type ChallengeStore struct {
	mu   sync.RWMutex
	data map[string]item
}

type item struct {
	value   []byte
	expires time.Time
}

// NewChallengeStore creates a new in-memory challenge store.
func NewChallengeStore() *ChallengeStore {
	return &ChallengeStore{data: make(map[string]item)}
}

var _ store.ChallengeStore = (*ChallengeStore)(nil)

func (s *ChallengeStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().UTC().Add(ttl)
	}
	s.data[key] = item{value: append([]byte(nil), value...), expires: exp}
	return nil
}

func (s *ChallengeStore) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	it, ok := s.data[key]
	s.mu.RUnlock()
	if !ok || (it.expires.IsZero() == false && time.Now().UTC().After(it.expires)) {
		return nil, store.ErrNotFound
	}
	return append([]byte(nil), it.value...), nil
}

func (s *ChallengeStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
	return nil
}

func (s *ChallengeStore) Take(ctx context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	it, ok := s.data[key]
	if ok {
		delete(s.data, key)
	}
	s.mu.Unlock()
	if !ok || (it.expires.IsZero() == false && time.Now().UTC().After(it.expires)) {
		return nil, nil
	}
	return append([]byte(nil), it.value...), nil
}

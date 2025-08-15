package memstore

import (
	"context"
	"sync"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

// KV is an in-memory KV store with TTL semantics evaluated at access time.
type KV struct {
	mu   sync.RWMutex
	data map[string]kvItem
}

type kvItem struct {
	value   []byte
	expires time.Time
}

// NewKV creates a new in-memory KV store.
func NewKV() *KV {
	return &KV{data: make(map[string]kvItem)}
}

var _ store.KV = (*KV)(nil)

func (s *KV) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().UTC().Add(ttl)
	}
	s.data[key] = kvItem{value: append([]byte(nil), value...), expires: exp}
	return nil
}

func (s *KV) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	it, ok := s.data[key]
	s.mu.RUnlock()
	if !ok || (it.expires.IsZero() == false && time.Now().UTC().After(it.expires)) {
		return nil, nil
	}
	return append([]byte(nil), it.value...), nil
}

func (s *KV) Del(ctx context.Context, key string) error {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
	return nil
}

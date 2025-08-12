package inmemory

import (
	"context"
	"sync"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
)

// AuditStore is an in-memory implementation of store.AuditStore.
type AuditStore struct {
	mu     sync.RWMutex
	events []domain.Event
}

// NewAuditStore creates a new in-memory audit store.
func NewAuditStore() *AuditStore {
	return &AuditStore{
		events: make([]domain.Event, 0),
	}
}

// Record stores an audit event.
func (s *AuditStore) Record(ctx context.Context, evt domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, evt)
	return nil
}

// GetEvents returns all audit events (for testing).
//
// This method is not part of the store.AuditStore interface but is useful for testing.
func (s *AuditStore) GetEvents() []domain.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy of the events
	result := make([]domain.Event, len(s.events))
	copy(result, s.events)
	return result
}
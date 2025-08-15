package inmemory

import (
	"context"
	"sync"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
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

	// Make a deep copy of the event to prevent external modifications
	eventCopy := evt
	
	// Deep copy the metadata map if it exists
	if evt.Metadata != nil {
		eventCopy.Metadata = make(map[string]interface{})
		for k, v := range evt.Metadata {
			eventCopy.Metadata[k] = v
		}
	}

	s.events = append(s.events, eventCopy)
	return nil
}

// GetEvents returns all audit events (for testing).
//
// This method is not part of the store.AuditStore interface but is useful for testing.
func (s *AuditStore) GetEvents() []domain.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a deep copy of the events
	result := make([]domain.Event, len(s.events))
	for i, evt := range s.events {
		result[i] = evt
		
		// Deep copy the metadata map if it exists
		if evt.Metadata != nil {
			result[i].Metadata = make(map[string]interface{})
			for k, v := range evt.Metadata {
				result[i].Metadata[k] = v
			}
		}
	}
	return result
}
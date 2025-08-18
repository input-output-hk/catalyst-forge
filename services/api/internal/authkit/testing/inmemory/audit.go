package inmemory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
)

// AuditStore is an in-memory implementation of store.AuditStore.
type AuditStore struct {
	mu     sync.RWMutex
	events []domain.Event
}

// NewAuditStore creates a new in-memory audit store.
func NewAuditStore() *AuditStore { return &AuditStore{events: make([]domain.Event, 0)} }

// Record stores an audit event.
func (s *AuditStore) Record(ctx context.Context, evt domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyEvt := evt
	if evt.Metadata != nil {
		copyEvt.Metadata = make(map[string]interface{})
		for k, v := range evt.Metadata {
			copyEvt.Metadata[k] = v
		}
	}
	s.events = append(s.events, copyEvt)
	return nil
}

// GetEvents returns all audit events (for testing).
func (s *AuditStore) GetEvents() []domain.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Event, len(s.events))
	for i, evt := range s.events {
		result[i] = evt
		if evt.Metadata != nil {
			result[i].Metadata = make(map[string]interface{})
			for k, v := range evt.Metadata {
				result[i].Metadata[k] = v
			}
		}
	}
	return result
}

// List returns audit events matching optional filters, ordered by CreatedAt desc.
func (s *AuditStore) List(_ context.Context, actorID *uuid.UUID, userID *uuid.UUID, types []string, since *time.Time, until *time.Time, limit, offset int) ([]domain.Event, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Build type set if provided
	typeSet := map[domain.EventType]struct{}{}
	for _, t := range types {
		typeSet[domain.EventType(t)] = struct{}{}
	}
	filtered := make([]domain.Event, 0, len(s.events))
	for _, e := range s.events {
		if actorID != nil && e.ActorID != nil && *e.ActorID != *actorID {
			continue
		}
		if userID != nil && e.UserID != nil && *e.UserID != *userID {
			continue
		}
		if len(typeSet) > 0 {
			if _, ok := typeSet[e.Type]; !ok {
				continue
			}
		}
		if since != nil && e.CreatedAt.Before(*since) {
			continue
		}
		if until != nil && e.CreatedAt.After(*until) {
			continue
		}
		// deep copy event and metadata
		copyEvt := e
		if e.Metadata != nil {
			copyEvt.Metadata = make(map[string]interface{})
			for k, v := range e.Metadata {
				copyEvt.Metadata[k] = v
			}
		}
		filtered = append(filtered, copyEvt)
	}
	// Order by CreatedAt desc
	for i := 0; i < len(filtered)-1; i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[i].CreatedAt.Before(filtered[j].CreatedAt) {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}
	total := int64(len(filtered))
	if offset > len(filtered) {
		return []domain.Event{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

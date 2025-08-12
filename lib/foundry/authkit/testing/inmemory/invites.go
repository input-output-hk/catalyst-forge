package inmemory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
	"github.com/google/uuid"
)

// InviteStore is an in-memory implementation of store.InviteStore.
type InviteStore struct {
	mu      sync.RWMutex
	invites map[uuid.UUID]*domain.Invite
}

// NewInviteStore creates a new in-memory invite store.
func NewInviteStore() *InviteStore {
	return &InviteStore{
		invites: make(map[uuid.UUID]*domain.Invite),
	}
}

// Create stores a new invite.
func (s *InviteStore) Create(ctx context.Context, inv *domain.Invite) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.invites[inv.ID]; exists {
		return errors.New("invite already exists")
	}

	// Store a copy
	invCopy := *inv
	s.invites[inv.ID] = &invCopy

	return nil
}

// Get retrieves an invite by its ID.
func (s *InviteStore) Get(ctx context.Context, id uuid.UUID) (*domain.Invite, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inv, exists := s.invites[id]
	if !exists {
		return nil, errors.New("invite not found")
	}

	// Return a copy
	invCopy := *inv
	return &invCopy, nil
}

// IncrementAttempts increments the failed attempt counter for an invite.
func (s *InviteStore) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, exists := s.invites[id]
	if !exists {
		return errors.New("invite not found")
	}

	inv.Attempts++

	return nil
}

// Redeem marks an invite as successfully redeemed.
func (s *InviteStore) Redeem(ctx context.Context, id uuid.UUID, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, exists := s.invites[id]
	if !exists {
		return errors.New("invite not found")
	}

	if inv.RedeemedAt != nil {
		return errors.New("invite already redeemed")
	}

	inv.RedeemedAt = &at

	return nil
}
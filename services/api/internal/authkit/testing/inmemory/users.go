package inmemory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/google/uuid"
)

// UserStore is an in-memory implementation of store.UserStore.
type UserStore struct {
	mu              sync.RWMutex
	usersByID       map[uuid.UUID]*domain.User
	usersByEmail    map[string]*domain.User
}

// NewUserStore creates a new in-memory user store.
func NewUserStore() *UserStore {
	return &UserStore{
		usersByID:    make(map[uuid.UUID]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
}

// Create creates a new user with the given email and roles.
func (s *UserStore) Create(ctx context.Context, email string, roles []string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user already exists
	if _, exists := s.usersByEmail[email]; exists {
		return nil, errors.New("user already exists")
	}

	now := time.Now()
	user := &domain.User{
		ID:             uuid.New(),
		Email:          email,
		Roles:          roles,
		SessionVersion: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	s.usersByID[user.ID] = user
	s.usersByEmail[user.Email] = user

	return user, nil
}

// GetByEmail retrieves a user by their email address.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.usersByEmail[email]
	if !exists {
		return nil, errors.New("user not found")
	}

	// Return a copy to prevent external modifications
	userCopy := *user
	return &userCopy, nil
}

// GetByID retrieves a user by their ID.
func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.usersByID[id]
	if !exists {
		return nil, errors.New("user not found")
	}

	// Return a copy to prevent external modifications
	userCopy := *user
	return &userCopy, nil
}

// UpdateRoles updates the roles for a user.
func (s *UserStore) UpdateRoles(ctx context.Context, id uuid.UUID, roles []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.usersByID[id]
	if !exists {
		return errors.New("user not found")
	}

	user.Roles = roles
	user.UpdatedAt = time.Now()

	return nil
}

// BumpSessionVersion increments the session version to invalidate all sessions.
func (s *UserStore) BumpSessionVersion(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.usersByID[id]
	if !exists {
		return errors.New("user not found")
	}

	user.SessionVersion++
	user.UpdatedAt = time.Now()

	return nil
}
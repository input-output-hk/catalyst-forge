package inmemory

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
)

// UserStore is an in-memory implementation of store.UserStore.
type UserStore struct {
	mu           sync.RWMutex
	usersByID    map[uuid.UUID]*domain.User
	usersByEmail map[string]*domain.User
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

	if _, exists := s.usersByEmail[email]; exists {
		return nil, errors.New("user already exists")
	}

	now := time.Now()
	user := &domain.User{ID: uuid.New(), Email: email, Roles: roles, SessionVersion: 1, CreatedAt: now, UpdatedAt: now}
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
	cp := *user
	return &cp, nil
}

// GetByID retrieves a user by their ID.
func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.usersByID[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	cp := *user
	return &cp, nil
}

// UpdateRoles updates the roles for a user.
func (s *UserStore) UpdateRoles(ctx context.Context, id uuid.UUID, roles []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, exists := s.usersByID[id]
	if !exists {
		return errors.New("user not found")
	}
	u.Roles = roles
	u.UpdatedAt = time.Now()
	return nil
}

// BumpSessionVersion increments the session version to invalidate all sessions.
func (s *UserStore) BumpSessionVersion(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, exists := s.usersByID[id]
	if !exists {
		return errors.New("user not found")
	}
	u.SessionVersion++
	u.UpdatedAt = time.Now()
	return nil
}

// UpdateFullName updates the user's display name.
func (s *UserStore) UpdateFullName(_ context.Context, id uuid.UUID, fullName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.usersByID[id]
	if !ok {
		return errors.New("user not found")
	}
	u.FullName = fullName
	u.UpdatedAt = time.Now()
	return nil
}

// List returns users ordered by creation time descending.
func (s *UserStore) List(_ context.Context, limit, offset int) ([]*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]*domain.User, 0, len(s.usersByID))
	for _, u := range s.usersByID {
		cp := *u
		all = append(all, &cp)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	if offset > len(all) {
		return []*domain.User{}, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

// ListFiltered filters by email substring and role.
func (s *UserStore) ListFiltered(_ context.Context, q string, role string, limit, offset int) ([]*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ql := strings.ToLower(q)
	out := make([]*domain.User, 0)
	for _, u := range s.usersByID {
		if q != "" && !strings.Contains(strings.ToLower(u.Email), ql) {
			continue
		}
		if role != "" {
			has := false
			for _, r := range u.Roles {
				if r == role {
					has = true
					break
				}
			}
			if !has {
				continue
			}
		}
		cp := *u
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if offset > len(out) {
		return []*domain.User{}, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

// CountFiltered returns the total number of users matching the filters.
func (s *UserStore) CountFiltered(_ context.Context, q string, role string) (int64, error) {
	list, _ := s.ListFiltered(context.Background(), q, role, 0, 0)
	return int64(len(list)), nil
}

// UpdateSuspended sets or clears the suspended_at timestamp.
func (s *UserStore) UpdateSuspended(_ context.Context, id uuid.UUID, suspended bool, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.usersByID[id]
	if !ok {
		return errors.New("user not found")
	}
	if suspended {
		u.SuspendedAt = &at
	} else {
		u.SuspendedAt = nil
	}
	u.UpdatedAt = time.Now()
	return nil
}

// Delete permanently removes a user.
func (s *UserStore) Delete(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.usersByID[id]
	if !ok {
		return errors.New("user not found")
	}
	delete(s.usersByEmail, u.Email)
	delete(s.usersByID, id)
	return nil
}

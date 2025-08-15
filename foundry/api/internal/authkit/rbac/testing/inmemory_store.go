package testing

import (
	"context"
	"sync"

	r "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/rbac"
	"github.com/google/uuid"
)

// InMemoryStore is a simple RBAC store for tests.
type InMemoryStore struct {
	mu       sync.RWMutex
	roles    map[string]r.RoleDef
	bindings []r.Binding
	versions map[string]int64 // key: subjectType+":"+subjectID
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		roles:    make(map[string]r.RoleDef),
		versions: make(map[string]int64),
	}
}

func (s *InMemoryStore) CreateRole(_ context.Context, role r.RoleDef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roles[role.Slug] = role
	return nil
}

func (s *InMemoryStore) GetRole(_ context.Context, slug string) (*r.RoleDef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	role, ok := s.roles[slug]
	if !ok {
		return nil, nil
	}
	cp := role
	return &cp, nil
}

func (s *InMemoryStore) UpdateRole(_ context.Context, role r.RoleDef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roles[role.Slug] = role
	return nil
}

func (s *InMemoryStore) ListRoles(_ context.Context) ([]r.RoleDef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]r.RoleDef, 0, len(s.roles))
	for _, v := range s.roles {
		out = append(out, v)
	}
	return out, nil
}

func (s *InMemoryStore) BumpRoleVersion(_ context.Context, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	role := s.roles[slug]
	role.Version++
	s.roles[slug] = role
	return nil
}

func (s *InMemoryStore) AddBinding(_ context.Context, b r.Binding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bindings = append(s.bindings, b)
	key := string(b.Subject.Type) + ":" + b.Subject.ID
	s.versions[key]++
	return nil
}

func (s *InMemoryStore) RemoveBinding(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.bindings[:0]
	for _, b := range s.bindings {
		if b.ID != id {
			next = append(next, b)
		} else {
			key := string(b.Subject.Type) + ":" + b.Subject.ID
			s.versions[key]++
		}
	}
	s.bindings = next
	return nil
}

func (s *InMemoryStore) ListBindings(_ context.Context, subj r.Subject) ([]r.Binding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []r.Binding
	for _, b := range s.bindings {
		if b.Subject.Type == subj.Type && b.Subject.ID == subj.ID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (s *InMemoryStore) ListBindingsByScope(_ context.Context, scope r.ScopeType, scopeID string) ([]r.Binding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []r.Binding
	for _, b := range s.bindings {
		if b.ScopeType == scope && b.ScopeID == scopeID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (s *InMemoryStore) GetPrincipalVersion(_ context.Context, subj r.Subject) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.versions[string(subj.Type)+":"+subj.ID], nil
}

func (s *InMemoryStore) BumpPrincipalVersion(_ context.Context, subj r.Subject) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.versions[string(subj.Type)+":"+subj.ID]++
	return nil
}

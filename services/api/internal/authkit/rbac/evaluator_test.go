package rbac

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClock struct{ t time.Time }

func (f fakeClock) Now() time.Time { return f.t }

type fakeLogger struct{}

func (fakeLogger) Debug(ctx context.Context, msg string, fields ...any) {}
func (fakeLogger) Info(ctx context.Context, msg string, fields ...any)  {}
func (fakeLogger) Warn(ctx context.Context, msg string, fields ...any)  {}
func (fakeLogger) Error(ctx context.Context, msg string, fields ...any) {}

type fakeStore struct {
	mu       sync.RWMutex
	roles    map[string]RoleDef
	bindings []Binding
	versions map[string]int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{roles: make(map[string]RoleDef), versions: make(map[string]int64)}
}

func (s *fakeStore) CreateRole(_ context.Context, role RoleDef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roles[role.Slug] = role
	return nil
}
func (s *fakeStore) GetRole(_ context.Context, slug string) (*RoleDef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.roles[slug]
	if !ok {
		return nil, nil
	}
	cp := r
	return &cp, nil
}
func (s *fakeStore) UpdateRole(_ context.Context, role RoleDef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roles[role.Slug] = role
	return nil
}
func (s *fakeStore) ListRoles(_ context.Context) ([]RoleDef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]RoleDef, 0, len(s.roles))
	for _, v := range s.roles {
		out = append(out, v)
	}
	return out, nil
}
func (s *fakeStore) BumpRoleVersion(_ context.Context, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.roles[slug]
	r.Version++
	s.roles[slug] = r
	return nil
}
func (s *fakeStore) AddBinding(_ context.Context, b Binding) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bindings = append(s.bindings, b)
	key := string(b.Subject.Type) + ":" + b.Subject.ID
	s.versions[key]++
	return nil
}
func (s *fakeStore) RemoveBinding(_ context.Context, id uuid.UUID) error {
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
func (s *fakeStore) ListBindings(_ context.Context, subj Subject) ([]Binding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Binding
	for _, b := range s.bindings {
		if b.Subject.Type == subj.Type && b.Subject.ID == subj.ID {
			out = append(out, b)
		}
	}
	return out, nil
}
func (s *fakeStore) ListBindingsByScope(_ context.Context, scope ScopeType, scopeID string) ([]Binding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Binding
	for _, b := range s.bindings {
		if b.ScopeType == scope && b.ScopeID == scopeID {
			out = append(out, b)
		}
	}
	return out, nil
}
func (s *fakeStore) GetPrincipalVersion(_ context.Context, subj Subject) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.versions[string(subj.Type)+":"+subj.ID], nil
}
func (s *fakeStore) BumpPrincipalVersion(_ context.Context, subj Subject) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.versions[string(subj.Type)+":"+subj.ID]++
	return nil
}

func TestEvaluator_DenyOverridesAllow(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	fixed := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)
	deps := Deps{Store: store, Clock: fakeClock{t: fixed}, Logger: fakeLogger{}, Cache: NewMemoryCache()}
	mgr := newManager(DefaultConfig(), deps)

	// Role with both allow and deny for same permission; deny should win
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{
		Slug: "ops",
		Entries: []RoleEntry{
			{Effect: Allow, Permission: "infra:deploy", ResourceType: "project"},
			{Effect: Deny, Permission: "infra:deploy", ResourceType: "project"},
		},
	}))

	subj := Subject{Type: SubjectUser, ID: "u1"}
	res := ResourceRef{Type: "project", ID: "p1"}
	// Bind subject to role
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "ops", ScopeType: ScopeProject, ScopeID: "p1"}))

	dec, err := mgr.Check(context.Background(), subj, PermissionKey("infra:deploy"), res)
	require.NoError(t, err)
	assert.Equal(t, DecisionDeny, dec)
}

func TestEvaluator_StepUpConditionMapsError(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	fixed := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)
	deps := Deps{Store: store, Clock: fakeClock{t: fixed}, Logger: fakeLogger{}, Cache: NewMemoryCache()}
	mgr := newManager(DefaultConfig(), deps)

	require.NoError(t, store.CreateRole(context.Background(), RoleDef{
		Slug: "ops",
		Entries: []RoleEntry{{
			Effect: Allow, Permission: "infra:deploy", ResourceType: "project",
			Conditions: []Condition{{Name: "requires_step_up", Params: map[string]any{"within": "5m"}}},
		}},
	}))

	subj := Subject{Type: SubjectUser, ID: "u1"}
	res := ResourceRef{Type: "project", ID: "p1"}
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "ops", ScopeType: ScopeProject, ScopeID: "p1"}))

	dec, err := mgr.Check(context.Background(), subj, PermissionKey("infra:deploy"), res)
	assert.Equal(t, DecisionDeny, dec)
	assert.ErrorIs(t, err, ErrConditionStepUpRequired)
}

func TestEvaluator_RoleCacheInvalidation(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	fixed := time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)
	deps := Deps{Store: store, Clock: fakeClock{t: fixed}, Logger: fakeLogger{}, Cache: NewMemoryCache()}
	mgr := newManager(DefaultConfig(), deps)

	// Initial role: deny deploy
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{
		Slug:    "ops",
		Version: 1,
		Entries: []RoleEntry{{Effect: Deny, Permission: "infra:deploy", ResourceType: "project"}},
	}))
	subj := Subject{Type: SubjectUser, ID: "u1"}
	res := ResourceRef{Type: "project", ID: "p1"}
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "ops", ScopeType: ScopeProject, ScopeID: "p1"}))

	dec, err := mgr.Check(context.Background(), subj, PermissionKey("infra:deploy"), res)
	require.NoError(t, err)
	assert.Equal(t, DecisionDeny, dec)

	// Update role to allow and bump version
	require.NoError(t, store.UpdateRole(context.Background(), RoleDef{
		Slug:    "ops",
		Version: 2,
		Entries: []RoleEntry{{Effect: Allow, Permission: "infra:deploy", ResourceType: "project"}},
	}))

	dec2, err := mgr.Check(context.Background(), subj, PermissionKey("infra:deploy"), res)
	require.NoError(t, err)
	assert.Equal(t, DecisionAllow, dec2)
}

func TestEvaluator_ResourceAgnosticNoResolver(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	deps := Deps{Store: store, Clock: fakeClock{t: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)}, Logger: fakeLogger{}, Cache: NewMemoryCache()}
	mgr := newManager(DefaultConfig(), deps)

	// role grants permission with no resource type (resource-agnostic)
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "reader", Entries: []RoleEntry{{Effect: Allow, Permission: "foo:read", ResourceType: ""}}}))
	subj := Subject{Type: SubjectUser, ID: "u1"}
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "reader", ScopeType: ScopeGlobal}))

	dec, err := mgr.Check(context.Background(), subj, PermissionKey("foo:read"), ResourceRef{Type: "project", ID: "p1"})
	require.NoError(t, err)
	assert.Equal(t, DecisionAllow, dec)
}

func TestEvaluator_ScopePrecedence_DenyAtHigherScope(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	deps := Deps{Store: store, Clock: fakeClock{t: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)}, Logger: fakeLogger{}, Cache: NewMemoryCache()}
	mgr := newManager(DefaultConfig(), deps)

	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "global-deny", Entries: []RoleEntry{{Effect: Deny, Permission: "data:read"}}}))
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "project-allow", Entries: []RoleEntry{{Effect: Allow, Permission: "data:read", ResourceType: "project"}}}))

	subj := Subject{Type: SubjectUser, ID: "u1"}
	// Bind both roles (simulate precedence). For now evaluator doesn't implement explicit scope ordering; this test will lock in deny-first semantics regardless of scope tagging.
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "global-deny", ScopeType: ScopeGlobal}))
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "project-allow", ScopeType: ScopeProject, ScopeID: "p1"}))

	dec, err := mgr.Check(context.Background(), subj, PermissionKey("data:read"), ResourceRef{Type: "project", ID: "p1"})
	require.NoError(t, err)
	assert.Equal(t, DecisionDeny, dec)
}

func TestEvaluator_ScopePrecedence_AllowSpecificityOrder(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	deps := Deps{Store: store, Clock: fakeClock{t: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)}, Logger: fakeLogger{}, Cache: NewMemoryCache()}
	mgr := newManager(DefaultConfig(), deps)

	// Global allow but project deny should still deny (deny first)
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "global-allow", Entries: []RoleEntry{{Effect: Allow, Permission: "data:read"}}}))
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "project-deny", Entries: []RoleEntry{{Effect: Deny, Permission: "data:read", ResourceType: "project"}}}))
	subj := Subject{Type: SubjectUser, ID: "u1"}
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "global-allow", ScopeType: ScopeGlobal}))
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "project-deny", ScopeType: ScopeProject, ScopeID: "p1"}))
	dec, err := mgr.Check(context.Background(), subj, PermissionKey("data:read"), ResourceRef{Type: "project", ID: "p1"})
	require.NoError(t, err)
	assert.Equal(t, DecisionDeny, dec)

	// Now flip: project allow + global empty -> allow
	store = newFakeStore()
	deps = Deps{Store: store, Clock: fakeClock{t: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)}, Logger: fakeLogger{}, Cache: NewMemoryCache()}
	mgr = newManager(DefaultConfig(), deps)
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "project-allow", Entries: []RoleEntry{{Effect: Allow, Permission: "data:read", ResourceType: "project"}}}))
	subj = Subject{Type: SubjectUser, ID: "u1"}
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "project-allow", ScopeType: ScopeProject, ScopeID: "p1"}))
	dec2, err := mgr.Check(context.Background(), subj, PermissionKey("data:read"), ResourceRef{Type: "project", ID: "p1"})
	require.NoError(t, err)
	assert.Equal(t, DecisionAllow, dec2)
}

func TestEvaluator_PrincipalCacheHitAndMiss(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	deps := Deps{Store: store, Clock: fakeClock{t: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)}, Logger: fakeLogger{}, Cache: NewMemoryCache()}
	mgr := newManager(DefaultConfig(), deps)

	// Setup role that allows action
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "reader", Entries: []RoleEntry{{Effect: Allow, Permission: "data:read", ResourceType: "project"}}}))
	subj := Subject{Type: SubjectUser, ID: "u1"}
	res := ResourceRef{Type: "project", ID: "p1"}
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "reader", ScopeType: ScopeProject, ScopeID: "p1"}))

	// First check: miss principal cache -> allow
	dec1, err := mgr.Check(context.Background(), subj, PermissionKey("data:read"), res)
	require.NoError(t, err)
	assert.Equal(t, DecisionAllow, dec1)

	// Bump principal version (e.g., binding change) -> should invalidate and still allow after re-eval
	require.NoError(t, store.BumpPrincipalVersion(context.Background(), subj))
	dec2, err := mgr.Check(context.Background(), subj, PermissionKey("data:read"), res)
	require.NoError(t, err)
	assert.Equal(t, DecisionAllow, dec2)
}

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

type noopLogger struct{}

func (noopLogger) Debug(ctx context.Context, msg string, fields ...any) {}
func (noopLogger) Info(ctx context.Context, msg string, fields ...any)  {}
func (noopLogger) Warn(ctx context.Context, msg string, fields ...any)  {}
func (noopLogger) Error(ctx context.Context, msg string, fields ...any) {}

// minimal fake store (copied pattern from rbac tests)
type fakeRBACStore struct {
	roles    map[string]rbac.RoleDef
	bindings []rbac.Binding
}

func newFakeRBACStore() *fakeRBACStore { return &fakeRBACStore{roles: map[string]rbac.RoleDef{}} }
func (s *fakeRBACStore) CreateRole(_ context.Context, role rbac.RoleDef) error {
	s.roles[role.Slug] = role
	return nil
}
func (s *fakeRBACStore) GetRole(_ context.Context, slug string) (*rbac.RoleDef, error) {
	r, ok := s.roles[slug]
	if !ok {
		return nil, nil
	}
	cp := r
	return &cp, nil
}
func (s *fakeRBACStore) UpdateRole(_ context.Context, role rbac.RoleDef) error {
	s.roles[role.Slug] = role
	return nil
}
func (s *fakeRBACStore) ListRoles(_ context.Context) ([]rbac.RoleDef, error) {
	out := make([]rbac.RoleDef, 0, len(s.roles))
	for _, v := range s.roles {
		out = append(out, v)
	}
	return out, nil
}
func (s *fakeRBACStore) BumpRoleVersion(_ context.Context, slug string) error {
	r := s.roles[slug]
	r.Version++
	s.roles[slug] = r
	return nil
}
func (s *fakeRBACStore) AddBinding(_ context.Context, b rbac.Binding) error {
	s.bindings = append(s.bindings, b)
	return nil
}
func (s *fakeRBACStore) RemoveBinding(_ context.Context, id uuid.UUID) error { return nil }
func (s *fakeRBACStore) ListBindings(_ context.Context, subj rbac.Subject) ([]rbac.Binding, error) {
	var out []rbac.Binding
	for _, b := range s.bindings {
		if b.Subject.Type == subj.Type && b.Subject.ID == subj.ID {
			out = append(out, b)
		}
	}
	return out, nil
}
func (s *fakeRBACStore) ListBindingsByScope(_ context.Context, scope rbac.ScopeType, scopeID string) ([]rbac.Binding, error) {
	return nil, nil
}
func (s *fakeRBACStore) GetPrincipalVersion(_ context.Context, subj rbac.Subject) (int64, error) {
	return 0, nil
}
func (s *fakeRBACStore) BumpPrincipalVersion(_ context.Context, subj rbac.Subject) error { return nil }

func TestPolicyEnforcer_RBAC_PermissionsAndStepUp(t *testing.T) {
	t.Parallel()

	// setup RBAC
	store := newFakeRBACStore()
	deps := rbac.Deps{Store: store, Clock: fixedClock{t: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)}, Logger: noopLogger{}, Cache: rbac.NewMemoryCache()}
	mgr, err := rbac.New(rbac.DefaultConfig(), deps)
	require.NoError(t, err)

	// role requiring deploy permission
	require.NoError(t, store.CreateRole(context.Background(), rbac.RoleDef{Slug: "ops", Entries: []rbac.RoleEntry{{Effect: rbac.Allow, Permission: "infra:deploy", ResourceType: "project"}}}))

	userID := uuid.New()
	subj := rbac.Subject{Type: rbac.SubjectUser, ID: userID.String()}
	require.NoError(t, store.AddBinding(context.Background(), rbac.Binding{ID: uuid.New(), Subject: subj, RoleSlug: "ops", ScopeType: rbac.ScopeProject, ScopeID: "p1"}))

	// Register resolver so RBAC can derive a project resource for /deploy
	mgr.RegisterResolver("/deploy", func(c *gin.Context) (rbac.ResourceRef, error) {
		return rbac.ResourceRef{Type: "project", ID: "p1"}, nil
	})
	// gin setup
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// attach AuthContext
	r.Use(func(c *gin.Context) {
		authkit.AuthContext{UserID: userID, Roles: []string{"ops"}}.Set(c)
		c.Next()
	})

	reg := authkit.NewPolicyRegistry().RequirePermissions([]string{"infra:deploy"}, "POST", "/deploy")
	enforcer := NewPolicyEnforcer(reg).WithRBAC(mgr)
	r.POST("/deploy", enforcer.EnforcePolicies(), func(c *gin.Context) { c.Status(http.StatusOK) })

	// 1) Forbidden without step-up when policy also requires step-up
	// Update registry to require step-up and ensure middleware returns 428
	reg.RequireStepUp("POST", "/deploy")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/deploy", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusPreconditionRequired, w.Code)

	// 2) Without step-up requirement (remove it), permission should allow
	reg = authkit.NewPolicyRegistry().RequirePermissions([]string{"infra:deploy"}, "POST", "/deploy")
	enforcer = NewPolicyEnforcer(reg).WithRBAC(mgr)
	r = gin.New()
	r.Use(func(c *gin.Context) { authkit.AuthContext{UserID: userID, Roles: []string{"ops"}}.Set(c); c.Next() })
	r.POST("/deploy", enforcer.EnforcePolicies(), func(c *gin.Context) { c.Status(http.StatusOK) })
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/deploy", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestPolicyEnforcer_RBAC_StepUpFromCondition(t *testing.T) {
	t.Parallel()

	store := newFakeRBACStore()
	deps := rbac.Deps{Store: store, Clock: fixedClock{t: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)}, Logger: noopLogger{}, Cache: rbac.NewMemoryCache()}
	mgr, err := rbac.New(rbac.DefaultConfig(), deps)
	require.NoError(t, err)

	// Role requires step-up via condition
	require.NoError(t, store.CreateRole(context.Background(), rbac.RoleDef{Slug: "ops", Entries: []rbac.RoleEntry{{
		Effect: rbac.Allow, Permission: "secure:write",
		Conditions: []rbac.Condition{{Name: "requires_step_up", Params: map[string]any{"within": "5m"}}},
	}}}))

	userID := uuid.New()
	subj := rbac.Subject{Type: rbac.SubjectUser, ID: userID.String()}
	require.NoError(t, store.AddBinding(context.Background(), rbac.Binding{ID: uuid.New(), Subject: subj, RoleSlug: "ops", ScopeType: rbac.ScopeGlobal}))

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Missing step-up -> expect 428
	r.Use(func(c *gin.Context) { authkit.AuthContext{UserID: userID}.Set(c); c.Next() })

	reg := authkit.NewPolicyRegistry().RequirePermissions([]string{"secure:write"}, "POST", "/secure")
	enforcer := NewPolicyEnforcer(reg).WithRBAC(mgr)
	r.POST("/secure", enforcer.EnforcePolicies(), func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/secure", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusPreconditionRequired, w.Code)
}

func TestPolicyEnforcer_RBAC_ForbiddenOnMissingPermission(t *testing.T) {
	t.Parallel()

	store := newFakeRBACStore()
	deps := rbac.Deps{Store: store, Clock: fixedClock{t: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)}, Logger: noopLogger{}, Cache: rbac.NewMemoryCache()}
	mgr, err := rbac.New(rbac.DefaultConfig(), deps)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Authenticated user without permission/binding
	userID := uuid.New()
	r.Use(func(c *gin.Context) { authkit.AuthContext{UserID: userID}.Set(c); c.Next() })

	reg := authkit.NewPolicyRegistry().RequirePermissions([]string{"secret:read"}, "GET", "/secret")
	enforcer := NewPolicyEnforcer(reg).WithRBAC(mgr)
	r.GET("/secret", enforcer.EnforcePolicies(), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/secret", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPolicyEnforcer_FallbackToAuthContextPermissions_WhenRBACNil(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	// Case 1: user has permission via AuthContext -> 200
	userID := uuid.New()
	r := gin.New()
	r.Use(func(c *gin.Context) {
		authkit.AuthContext{UserID: userID, Permissions: []string{"perm:x"}}.Set(c)
		c.Next()
	})
	reg := authkit.NewPolicyRegistry().RequirePermissions([]string{"perm:x"}, "GET", "/x")
	enforcer := NewPolicyEnforcer(reg) // RBAC manager not attached
	r.GET("/x", enforcer.EnforcePolicies(), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Case 2: user lacks permission -> 403
	r2 := gin.New()
	r2.Use(func(c *gin.Context) { authkit.AuthContext{UserID: userID}.Set(c); c.Next() })
	r2.GET("/x", enforcer.EnforcePolicies(), func(c *gin.Context) { c.Status(http.StatusOK) })
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/x", nil)
	r2.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code)
}

func TestPolicyEnforcer_RBAC_ResolverAndAttrCondition(t *testing.T) {
	t.Parallel()

	// RBAC setup with attr_equals condition
	store := newFakeRBACStore()
	deps := rbac.Deps{Store: store, Clock: fixedClock{t: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC)}, Logger: noopLogger{}, Cache: rbac.NewMemoryCache()}
	mgr, err := rbac.New(rbac.DefaultConfig(), deps)
	require.NoError(t, err)

	require.NoError(t, store.CreateRole(context.Background(), rbac.RoleDef{Slug: "owner", Entries: []rbac.RoleEntry{{
		Effect: rbac.Allow, Permission: "doc:read", ResourceType: "doc",
		Conditions: []rbac.Condition{{Name: "attr_equals", Params: map[string]any{"attr": "owner_id", "subject_field": "id"}}},
	}}}))

	userID := uuid.New()
	subj := rbac.Subject{Type: rbac.SubjectUser, ID: userID.String()}
	require.NoError(t, store.AddBinding(context.Background(), rbac.Binding{ID: uuid.New(), Subject: subj, RoleSlug: "owner", ScopeType: rbac.ScopeGlobal}))

	// Gin + policy
	gin.SetMode(gin.TestMode)
	r := gin.New()
	reg := authkit.NewPolicyRegistry().RequirePermissions([]string{"doc:read"}, "GET", "/doc").RequirePermissions([]string{"doc:read"}, "GET", "/doc2")
	enforcer := NewPolicyEnforcer(reg).WithRBAC(mgr)

	// Register exact-path resolvers
	mgr.RegisterResolver("/doc", func(c *gin.Context) (rbac.ResourceRef, error) {
		return rbac.ResourceRef{Type: "doc", ID: "d1", Attrs: map[string]any{"owner_id": userID.String()}}, nil
	})
	mgr.RegisterResolver("/doc2", func(c *gin.Context) (rbac.ResourceRef, error) {
		return rbac.ResourceRef{Type: "doc", ID: "d2", Attrs: map[string]any{"owner_id": uuid.New().String()}}, nil
	})

	// Attach auth context
	r.Use(func(c *gin.Context) { authkit.AuthContext{UserID: userID}.Set(c); c.Next() })
	r.GET("/doc", enforcer.EnforcePolicies(), func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/doc2", enforcer.EnforcePolicies(), func(c *gin.Context) { c.Status(http.StatusOK) })

	// Owner case -> 200
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/doc", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Not owner -> 403
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/doc2", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code)
}

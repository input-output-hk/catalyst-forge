package rbac

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedPlanner struct{ order []ScopeType }

func (p *fixedPlanner) Resolve(res ResourceRef) ([]ScopeType, map[ScopeType]string) {
	ids := make(map[ScopeType]string, len(p.order))
	for _, s := range p.order {
		if spec, ok := GetScopeSpec(s); ok {
			if id, okID := spec.DefaultExtractor(res); okID {
				ids[s] = id
			}
		}
	}
	return p.order, ids
}

func TestEvaluator_AllowOrder_RespectsPlannerOrder(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	deps := Deps{Store: store, Cache: NewMemoryCache()}
	cfg := DefaultConfig()
	// Prefer org before project
	cfg.Scopes = &fixedPlanner{order: []ScopeType{ScopeOrg, ScopeProject, ScopeRes, ScopeGlobal}}
	mgr := newManager(cfg, deps)

	// Role allowing data:read on project resources
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "reader", Entries: []RoleEntry{{Effect: Allow, Permission: "data:read", ResourceType: "project"}}}))
	subj := Subject{Type: SubjectUser, ID: "u1"}
	// Bind at both org and project
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "reader", ScopeType: ScopeOrg, ScopeID: "org-1"}))
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "reader", ScopeType: ScopeProject, ScopeID: "proj-9"}))

	// Resource: project under org
	res := BuildAncestryRef("org-1", "proj-9", "", "project", "proj-9")
	dec, trace, err := mgr.Explain(context.Background(), subj, PermissionKey("data:read"), res)
	require.NoError(t, err)
	require.Equal(t, DecisionAllow, dec)
	// First evaluated step should reflect planner's first scope (org)
	require.NotEmpty(t, trace.Steps)
	assert.Equal(t, ScopeOrg, trace.Steps[0].Scope)
	assert.Equal(t, "org-1", trace.Steps[0].ScopeID)
}

func TestEvaluator_DenyPrecedence_AcrossCustomChain(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	deps := Deps{Store: store, Cache: NewMemoryCache()}
	cfg := DefaultConfig()
	cfg.Scopes = &fixedPlanner{order: []ScopeType{ScopeRes, ScopeProject, ScopeOrg, ScopeGlobal}}
	mgr := newManager(cfg, deps)

	// Role entries for same permission; allow (resource), deny (org)
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "ops", Entries: []RoleEntry{{Effect: Allow, Permission: "deploy:run", ResourceType: "project"}}}))
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "controls", Entries: []RoleEntry{{Effect: Deny, Permission: "deploy:run"}}}))

	subj := Subject{Type: SubjectUser, ID: "u1"}
	// Bind allow at project, deny at org
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "ops", ScopeType: ScopeProject, ScopeID: "p1"}))
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "controls", ScopeType: ScopeOrg, ScopeID: "org-1"}))

	res := BuildAncestryRef("org-1", "p1", "", "project", "p1")
	dec, err := mgr.Check(context.Background(), subj, PermissionKey("deploy:run"), res)
	require.NoError(t, err)
	assert.Equal(t, DecisionDeny, dec)
}

func TestEvaluator_PathAwarePlanner_OverridePrecedence(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	deps := Deps{Store: store, Cache: NewMemoryCache()}
	base := NewDefaultPlanner()
	// Override for GET /api/v1/items to prioritize org
	p := NewPathAwarePlanner(base, []PathOverride{{PathPattern: `^/api/v1/items$`, Method: "GET", Chain: []ScopeType{ScopeOrg, ScopeProject, ScopeRes, ScopeGlobal}}})
	cfg := DefaultConfig()
	cfg.Scopes = p
	mgr := newManager(cfg, deps)

	// Role allowing read at project scope
	require.NoError(t, store.CreateRole(context.Background(), RoleDef{Slug: "reader", Entries: []RoleEntry{{Effect: Allow, Permission: "items:read", ResourceType: "project"}}}))
	subj := Subject{Type: SubjectUser, ID: "u1"}
	// Bind at both org and project
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "reader", ScopeType: ScopeOrg, ScopeID: "org-9"}))
	require.NoError(t, store.AddBinding(context.Background(), Binding{ID: uuid.New(), Subject: subj, RoleSlug: "reader", ScopeType: ScopeProject, ScopeID: "proj-7"}))

	res := BuildAncestryRef("org-9", "proj-7", "", "project", "proj-7")
	// Include request meta to trigger override
	ctx := WithRequestMeta(context.Background(), "/api/v1/items", "GET")
	_, trace, err := mgr.Explain(ctx, subj, PermissionKey("items:read"), res)
	require.NoError(t, err)
	require.NotEmpty(t, trace.Steps)
	assert.Equal(t, ScopeOrg, trace.Steps[0].Scope)
	assert.Equal(t, "org-9", trace.Steps[0].ScopeID)
}

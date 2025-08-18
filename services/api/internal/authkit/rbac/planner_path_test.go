package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathAwarePlanner_OverrideByPath(t *testing.T) {
	// Parent not parallelized as this file uses only pure functions and local vars
	// but keep consistency with other tests
	t.Parallel()

	base := NewDefaultPlanner()
	p := NewPathAwarePlanner(base, []PathOverride{
		{PathPattern: `^/api/v1/admin/.*$`, Method: "", Chain: []ScopeType{ScopeRes, ScopeOrg, ScopeGlobal}},
	})

	res := BuildAncestryRef("org-1", "proj-9", "env-7", "artifact", "a1")
	order, ids := p.ResolveWithMeta(res, map[string]string{"path": "/api/v1/admin/users", "method": "GET"})

	require.Equal(t, []ScopeType{ScopeRes, ScopeOrg, ScopeGlobal}, order)
	assert.Equal(t, "a1", ids[ScopeRes])
	assert.Equal(t, "org-1", ids[ScopeOrg])
	_, ok := ids[ScopeGlobal]
	assert.True(t, ok)
}

func TestPathAwarePlanner_MethodMatching_CaseInsensitive(t *testing.T) {
	t.Parallel()

	base := NewDefaultPlanner()
	p := NewPathAwarePlanner(base, []PathOverride{
		{PathPattern: `^/api/v1/deployments$`, Method: "POST", Chain: []ScopeType{ScopeRes, ScopeProject, ScopeOrg}},
	})

	res := BuildAncestryRef("org-1", "proj-9", "", "deployment", "dep-5")

	tests := []struct {
		name   string
		method string
		want   []ScopeType
	}{
		{name: "ok/lowercase-post-matches", method: "post", want: []ScopeType{ScopeRes, ScopeProject, ScopeOrg}},
		{name: "fallback/get-no-match", method: "GET", want: []ScopeType{ScopeRes, ScopeProject, ScopeOrg, ScopeGlobal}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			order, _ := p.ResolveWithMeta(res, map[string]string{"path": "/api/v1/deployments", "method": tc.method})
			require.Equal(t, tc.want, order)
		})
	}
}

func TestPathAwarePlanner_FirstMatchingOverrideWins(t *testing.T) {
	t.Parallel()

	base := NewDefaultPlanner()
	p := NewPathAwarePlanner(base, []PathOverride{
		{PathPattern: `^/api/v1/items$`, Method: "GET", Chain: []ScopeType{ScopeRes, ScopeOrg}},
		{PathPattern: `^/api/v1/items$`, Method: "GET", Chain: []ScopeType{ScopeRes, ScopeProject}},
	})

	res := BuildAncestryRef("org-1", "proj-9", "", "artifact", "a1")
	order, _ := p.ResolveWithMeta(res, map[string]string{"path": "/api/v1/items", "method": "GET"})
	// first override takes precedence
	require.Equal(t, []ScopeType{ScopeRes, ScopeOrg}, order)
}

func TestPathAwarePlanner_FallbackToBaseWhenNoOverride(t *testing.T) {
	t.Parallel()

	base := &RuleBasedScopePlanner{FallbackChain: []ScopeType{ScopeRes, ScopeEnvironment, ScopeProject, ScopeOrg}}
	p := NewPathAwarePlanner(base, nil)

	res := BuildAncestryRef("org-1", "proj-9", "env-7", "deployment", "dep-5")
	order, ids := p.ResolveWithMeta(res, map[string]string{"path": "/no/match", "method": "GET"})

	require.Equal(t, []ScopeType{ScopeRes, ScopeEnvironment, ScopeProject, ScopeOrg}, order)
	assert.Equal(t, "dep-5", ids[ScopeRes])
	assert.Equal(t, "env-7", ids[ScopeEnvironment])
	assert.Equal(t, "proj-9", ids[ScopeProject])
	assert.Equal(t, "org-1", ids[ScopeOrg])
}

func TestNewPathAwarePlanner_PanicsOnUnknownScope(t *testing.T) {
	t.Parallel()

	base := NewDefaultPlanner()
	bad := []PathOverride{{PathPattern: `^/x$`, Chain: []ScopeType{ScopeType("totally_unknown")}}}
	require.Panics(t, func() { _ = NewPathAwarePlanner(base, bad) }, "expected panic for unknown scope in override")
}

func TestNewPathAwarePlanner_PanicsOnInvalidRegex(t *testing.T) {
	t.Parallel()

	base := NewDefaultPlanner()
	bad := []PathOverride{{PathPattern: "(", Chain: []ScopeType{ScopeRes}}}
	require.Panics(t, func() { _ = NewPathAwarePlanner(base, bad) }, "expected panic for invalid regex")
}

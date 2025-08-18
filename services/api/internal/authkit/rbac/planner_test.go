package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper alias to shared test utils
func mkRef(orgID, projectID, envID, rtype, rid string) ResourceRef {
	return BuildAncestryRef(orgID, projectID, envID, rtype, rid)
}

func TestRuleBasedScopePlanner_FallbackChain(t *testing.T) {
	t.Parallel()

	planner := &RuleBasedScopePlanner{
		FallbackChain: []ScopeType{ScopeRes, ScopeProject, ScopeOrg, ScopeGlobal},
	}
	ref := mkRef("org-1", "proj-9", "", "deployment", "dep-5")
	order, ids := planner.Resolve(ref)

	require.Equal(t, []ScopeType{ScopeRes, ScopeProject, ScopeOrg, ScopeGlobal}, order)
	assert.Equal(t, "dep-5", ids[ScopeRes])
	assert.Equal(t, "proj-9", ids[ScopeProject])
	assert.Equal(t, "org-1", ids[ScopeOrg])
	// Global extractor returns empty string, but should still record presence
	_, ok := ids[ScopeGlobal]
	assert.True(t, ok, "global should be present in ids map")
}

func TestRuleBasedScopePlanner_RuleMatch(t *testing.T) {
	t.Parallel()

	planner := &RuleBasedScopePlanner{
		Rules: []ScopeChainRule{
			{ResourceTypePattern: "^artifact$", Chain: []ScopeType{ScopeRes, ScopeOrg, ScopeGlobal}},
		},
		FallbackChain: []ScopeType{ScopeRes, ScopeProject, ScopeOrg, ScopeGlobal},
	}
	ref := mkRef("org-1", "proj-1", "env-1", "artifact", "a1")
	order, ids := planner.Resolve(ref)

	require.Equal(t, []ScopeType{ScopeRes, ScopeOrg, ScopeGlobal}, order)
	assert.Equal(t, "a1", ids[ScopeRes])
	assert.Equal(t, "org-1", ids[ScopeOrg])
	_, ok := ids[ScopeGlobal]
	assert.True(t, ok)
}

func TestRuleBasedScopePlanner_Overrides(t *testing.T) {
	t.Parallel()

	over := func(stringToReturn string) IDExtractor {
		return func(ResourceRef) (string, bool) { return stringToReturn, true }
	}
	planner := &RuleBasedScopePlanner{
		Rules: []ScopeChainRule{
			{
				ResourceTypePattern: "^deployment$",
				Chain:               []ScopeType{ScopeRes, ScopeProject, ScopeOrg},
				Overrides:           map[ScopeType]IDExtractor{ScopeProject: over("OVERRIDE-PROJ")},
			},
		},
		FallbackChain: []ScopeType{ScopeRes, ScopeProject, ScopeOrg, ScopeGlobal},
	}
	ref := mkRef("org-1", "proj-9", "", "deployment", "dep-5")
	order, ids := planner.Resolve(ref)

	require.Equal(t, []ScopeType{ScopeRes, ScopeProject, ScopeOrg}, order)
	assert.Equal(t, "dep-5", ids[ScopeRes])
	assert.Equal(t, "OVERRIDE-PROJ", ids[ScopeProject])
	assert.Equal(t, "org-1", ids[ScopeOrg])
}

func TestRuleBasedScopePlanner_MissingIDs(t *testing.T) {
	t.Parallel()

	planner := &RuleBasedScopePlanner{
		FallbackChain: []ScopeType{ScopeRes, ScopeEnvironment, ScopeProject, ScopeOrg},
	}
	// No environment in ancestry, only project/org
	ref := mkRef("org-1", "proj-9", "", "deployment", "dep-5")
	order, ids := planner.Resolve(ref)

	require.Equal(t, []ScopeType{ScopeRes, ScopeEnvironment, ScopeProject, ScopeOrg}, order)
	assert.Equal(t, "dep-5", ids[ScopeRes])
	// env should be absent from ids when missing
	_, ok := ids[ScopeEnvironment]
	assert.False(t, ok, "environment should not be present in ids when missing from ancestry")
	assert.Equal(t, "proj-9", ids[ScopeProject])
	assert.Equal(t, "org-1", ids[ScopeOrg])
}

func TestRuleBasedScopePlanner_UnknownScopeIgnoredForIDs(t *testing.T) {
	t.Parallel()

	unknown := ScopeType("unknown_scope")
	planner := &RuleBasedScopePlanner{
		FallbackChain: []ScopeType{ScopeRes, unknown, ScopeOrg},
	}
	ref := mkRef("org-1", "", "", "artifact", "a1")
	order, ids := planner.Resolve(ref)

	// Order is preserved even with unknown; evaluator will ignore unknown later.
	require.Equal(t, []ScopeType{ScopeRes, unknown, ScopeOrg}, order)
	assert.Equal(t, "a1", ids[ScopeRes])
	assert.Equal(t, "org-1", ids[ScopeOrg])
	_, ok := ids[unknown]
	assert.False(t, ok, "unknown scope should not produce an ID entry")
}

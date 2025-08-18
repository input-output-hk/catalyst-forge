package planner

import rbac "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"

// Build constructs the application's ScopePlanner.
// It preserves default behavior for now; tasks 3 will add concrete rules/overrides.
func Build() rbac.ScopePlanner {
	base := &rbac.RuleBasedScopePlanner{
		Rules:         nil,
		FallbackChain: []rbac.ScopeType{rbac.ScopeRes, rbac.ScopeProject, rbac.ScopeOrg, rbac.ScopeGlobal},
	}
	// No path overrides yet; return base directly.
	return base
}

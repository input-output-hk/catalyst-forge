package policy

import (
	"encoding/json"
	"sync/atomic"

	libauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
)

// Provider implements libauth.PolicyProvider backed by an in-memory snapshot
// periodically refreshed from the repository. It prefers exact/prefix/regex
// semantics identical to libauth.PolicyRegistry by delegating to it.
type Provider struct {
	snapshot atomic.Value // *libauth.PolicyRegistry
}

// NewProvider builds a provider from a list of APIPolicy rows.
func NewProvider(policies []APIPolicy) *Provider {
	pr := &Provider{}
	pr.setSnapshot(policies)
	return pr
}

// setSnapshot compiles rules into a PolicyRegistry and swaps it atomically.
func (p *Provider) setSnapshot(policies []APIPolicy) {
	reg := libauth.NewPolicyRegistry()
	for _, row := range policies {
		if !row.Enabled {
			continue
		}
		if row.RequireAuth {
			reg.RequireAuth(row.Method, row.PathPattern)
		}
		if row.RequireStepUp {
			reg.RequireStepUp(row.Method, row.PathPattern)
		}
		// Roles and permissions: attach if present
		var roles []string
		var perms []string
		// We tolerate JSON decode failures by ignoring (admin UI should validate)
		_ = jsonUnmarshal(row.AllowedRoles, &roles)
		_ = jsonUnmarshal(row.AllowedPermissions, &perms)
		if len(roles) > 0 {
			reg.RequireRoles(roles, row.Method, row.PathPattern)
		}
		if len(perms) > 0 {
			reg.RequirePermissions(perms, row.Method, row.PathPattern)
		}
	}
	p.snapshot.Store(reg)
}

// GetRules returns compiled rules from the current snapshot.
func (p *Provider) GetRules(method string, path string) []libauth.Rule {
	reg, _ := p.snapshot.Load().(*libauth.PolicyRegistry)
	if reg == nil {
		return nil
	}
	return reg.GetRules(method, path)
}

// jsonUnmarshal is a tiny helper to decode optional datatypes.JSON into a slice.
func jsonUnmarshal(raw any, out *[]string) error {
	b, ok := raw.([]byte)
	if !ok || len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}

package rbac

import (
	"fmt"
	"regexp"
	"strings"
)

// PathOverride provides a path/method specific chain override.
// Method may be empty to match any method.
type PathOverride struct {
	PathPattern string
	Method      string
	Chain       []ScopeType
}

// PathAwarePlanner decorates a base ScopePlanner with path-based overrides.
// If a request meta matches any override, the override's Chain determines the
// order, and IDs are derived using registry/default extractors.
type PathAwarePlanner struct {
	Base      ScopePlanner
	Overrides []PathOverride
}

// NewPathAwarePlanner constructs a planner and validates override scope names.
// Panics on invalid configuration since these are programmer errors.
func NewPathAwarePlanner(base ScopePlanner, overrides []PathOverride) *PathAwarePlanner {
	// Validate that all scopes referenced by overrides are known
	for _, ov := range overrides {
		for _, s := range ov.Chain {
			if !IsKnownScope(s) {
				panic(fmt.Sprintf("rbac: unknown scope in override chain: %s", s))
			}
		}
		// Validate regex upfront
		if ov.PathPattern != "" {
			if _, err := regexp.Compile(ov.PathPattern); err != nil {
				panic(fmt.Sprintf("rbac: invalid path regex %q: %v", ov.PathPattern, err))
			}
		}
	}
	return &PathAwarePlanner{Base: base, Overrides: overrides}
}

// Resolve implements ScopePlanner by delegating to Base.
func (p *PathAwarePlanner) Resolve(res ResourceRef) ([]ScopeType, map[ScopeType]string) {
	return p.Base.Resolve(res)
}

// ResolveWithMeta implements ContextualScopePlanner.
// It applies the first matching override (by path regex and optional method),
// otherwise defers to Base.Resolve.
func (p *PathAwarePlanner) ResolveWithMeta(res ResourceRef, meta map[string]string) ([]ScopeType, map[ScopeType]string) {
	path := meta["path"]
	method := meta["method"]
	for _, ov := range p.Overrides {
		if ov.PathPattern == "" {
			continue
		}
		if ov.Method != "" && method != "" && !strings.EqualFold(ov.Method, method) {
			continue
		}
		matched, _ := regexp.MatchString(ov.PathPattern, path)
		if !matched {
			continue
		}
		// Build IDs from override chain using registry/default extractors
		ids := make(map[ScopeType]string, len(ov.Chain))
		for _, s := range ov.Chain {
			if spec, ok := GetScopeSpec(s); ok {
				if id, okID := spec.DefaultExtractor(res); okID {
					ids[s] = id
				}
			}
		}
		return ov.Chain, ids
	}
	return p.Base.Resolve(res)
}

package rbac

import (
	"regexp"
)

// ScopePlanner plans the evaluation order and resolves scope IDs for a resource.
// It returns the ordered list of scopes (most-specific to least-specific) and a
// map of resolved IDs for scopes that could be derived from the provided resource.
type ScopePlanner interface {
	Resolve(res ResourceRef) (order []ScopeType, ids map[ScopeType]string)
}

// ContextualScopePlanner optionally uses request metadata to determine scope order.
// Implementations may ignore meta keys they don't understand.
type ContextualScopePlanner interface {
	ResolveWithMeta(res ResourceRef, meta map[string]string) (order []ScopeType, ids map[ScopeType]string)
}

// ScopeChainRule selects a scope chain for a matching resource type, optionally
// providing extractor overrides per scope.
// ResourceTypePattern is a regular expression matched against ResourceRef.Type.
type ScopeChainRule struct {
	ResourceTypePattern string
	Chain               []ScopeType
	Overrides           map[ScopeType]IDExtractor
}

func (r ScopeChainRule) matches(resourceType string) bool {
	if r.ResourceTypePattern == "" {
		return false
	}
	ok, err := regexp.MatchString(r.ResourceTypePattern, resourceType)
	return err == nil && ok
}

// RuleBasedScopePlanner applies the first matching rule for a resource type.
// If no rules match, it uses FallbackChain.
type RuleBasedScopePlanner struct {
	Rules         []ScopeChainRule
	FallbackChain []ScopeType
}

// Resolve implements ScopePlanner.
func (p *RuleBasedScopePlanner) Resolve(res ResourceRef) ([]ScopeType, map[ScopeType]string) {
	order := p.FallbackChain
	for _, rule := range p.Rules {
		if rule.matches(res.Type) {
			order = rule.Chain
			break
		}
	}
	ids := make(map[ScopeType]string, len(order))
	for _, scope := range order {
		var extractor IDExtractor
		// Prefer rule-specific override if present; else use registry default.
		if ov := p.lookupOverride(scope); ov != nil {
			extractor = ov
		} else if spec, ok := GetScopeSpec(scope); ok {
			extractor = spec.DefaultExtractor
		} else {
			// Unknown scope names are ignored
			continue
		}
		if id, ok := extractor(res); ok {
			ids[scope] = id
		}
	}
	return order, ids
}

func (p *RuleBasedScopePlanner) lookupOverride(scope ScopeType) IDExtractor {
	for _, r := range p.Rules {
		if r.Overrides == nil {
			continue
		}
		if ov, ok := r.Overrides[scope]; ok {
			return ov
		}
	}
	return nil
}

// NewDefaultPlanner returns a planner that preserves the current behavior:
// resource -> project -> org -> global
func NewDefaultPlanner() ScopePlanner {
	return &RuleBasedScopePlanner{
		Rules:         nil,
		FallbackChain: []ScopeType{ScopeRes, ScopeProject, ScopeOrg, ScopeGlobal},
	}
}

package rbac

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ScopeType identifies the type of scope for a binding
type ScopeType string

// Scope constants - these must match values emitted by resolvers
const (
	ScopeGlobal      ScopeType = "global"
	ScopeOrg         ScopeType = "org"
	ScopeProject     ScopeType = "project"
	ScopeEnvironment ScopeType = "environment"
	ScopeRes         ScopeType = "resource"
)

// IDExtractor extracts a scope ID from a ResourceRef
type IDExtractor func(res ResourceRef) (string, bool)

// ScopeSpec defines a scope with its default extractor and specificity level
type ScopeSpec struct {
	Type             ScopeType
	DefaultExtractor IDExtractor
	Specificity      int // lower = more specific
}

// scopeRegistry maps known scope types to their specifications
var scopeRegistry = map[ScopeType]ScopeSpec{
	ScopeRes:         {ScopeRes, func(res ResourceRef) (string, bool) { return res.ID, res.ID != "" }, 0},
	ScopeEnvironment: {ScopeEnvironment, ParentTypeExtractor("environment"), 1},
	ScopeProject:     {ScopeProject, ParentTypeExtractor("project"), 2},
	ScopeOrg:         {ScopeOrg, OrgExtractor(), 3},
	ScopeGlobal:      {ScopeGlobal, func(ResourceRef) (string, bool) { return "", true }, 4},
}

// scopeMu guards access to scopeRegistry for dynamic registrations.
var scopeMu sync.RWMutex

// ParentTypeExtractor creates an extractor that finds a parent resource of the specified type
func ParentTypeExtractor(target string) IDExtractor {
	return func(res ResourceRef) (string, bool) {
		for p := &res; p != nil; p = p.Parent {
			if p.Type == target && p.ID != "" {
				return p.ID, true
			}
			if p.Parent == nil {
				break
			}
		}
		return "", false
	}
}

// OrgExtractor extracts the organization ID from a resource
func OrgExtractor() IDExtractor {
	return func(res ResourceRef) (string, bool) {
		if res.OrgID != nil {
			return res.OrgID.String(), true
		}
		for p := &res; p != nil; p = p.Parent {
			if p.Type == "org" && p.ID != "" {
				return p.ID, true
			}
			if p.Parent == nil {
				break
			}
		}
		return "", false
	}
}

// IsKnownScope checks if a scope type is registered in the scope registry
func IsKnownScope(scopeType ScopeType) bool {
	scopeMu.RLock()
	defer scopeMu.RUnlock()
	_, exists := scopeRegistry[scopeType]
	return exists
}

// ValidateScopeNames validates that all provided scope names exist in the registry
func ValidateScopeNames(names ...ScopeType) error {
	var unknown []string
	for _, name := range names {
		if !IsKnownScope(name) {
			unknown = append(unknown, string(name))
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("unknown scope types: %s", strings.Join(unknown, ", "))
	}
	return nil
}

// GetScopeSpec returns the specification for a given scope type
func GetScopeSpec(scopeType ScopeType) (ScopeSpec, bool) {
	scopeMu.RLock()
	defer scopeMu.RUnlock()
	spec, exists := scopeRegistry[scopeType]
	return spec, exists
}

// ListKnownScopes returns all known scope types in sorted order
func ListKnownScopes() []ScopeType {
	scopeMu.RLock()
	defer scopeMu.RUnlock()
	scopes := make([]ScopeType, 0, len(scopeRegistry))
	for scopeType := range scopeRegistry {
		scopes = append(scopes, scopeType)
	}
	sort.Slice(scopes, func(i, j int) bool {
		return scopeRegistry[scopes[i]].Specificity < scopeRegistry[scopes[j]].Specificity
	})
	return scopes
}

// RegisterScope registers a new scope in the central registry.
// Returns an error if the scope type is empty, the extractor is nil,
// or the scope is already registered.
func RegisterScope(spec ScopeSpec) error {
	if strings.TrimSpace(string(spec.Type)) == "" {
		return fmt.Errorf("rbac: scope Type must be non-empty")
	}
	if spec.DefaultExtractor == nil {
		return fmt.Errorf("rbac: scope %q missing DefaultExtractor", spec.Type)
	}
	if spec.Specificity < 0 {
		return fmt.Errorf("rbac: scope %q has invalid Specificity %d", spec.Type, spec.Specificity)
	}
	scopeMu.Lock()
	defer scopeMu.Unlock()
	if _, exists := scopeRegistry[spec.Type]; exists {
		return fmt.Errorf("rbac: scope %q already registered", spec.Type)
	}
	scopeRegistry[spec.Type] = spec
	return nil
}

package authkit

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule defines authorization requirements for a path pattern.
type Rule struct {
	RequireAuth   bool     // Requires authentication
	RequireStepUp bool     // Requires recent WebAuthn authentication
	Roles         []string // Any-of semantics - user must have at least one role
	Permissions   []string // Any-of semantics - user must have at least one permission
}

// PolicyRegistry manages path-based authorization rules.
type PolicyRegistry struct {
	rules map[string][]compiledRule // method -> list of matchers
}

// compiledRule is an internal representation of a rule with its matcher.
type compiledRule struct {
	pattern string
	matcher pathMatcher
	rule    Rule
}

// pathMatcher is an interface for different types of path matching.
type pathMatcher interface {
	// Match returns true if the given path matches the pattern.
	Match(path string) bool
}

// exactMatcher matches exact paths.
type exactMatcher struct {
	path string
}

// Match returns true if the path exactly matches.
func (m exactMatcher) Match(path string) bool {
	return path == m.path
}

// prefixMatcher matches path prefixes (for wildcard patterns).
type prefixMatcher struct {
	prefix string
}

// Match returns true if the path starts with the prefix and respects segment boundaries.
func (m prefixMatcher) Match(path string) bool {
	if path == m.prefix {
		return true // allow exact match
	}
	if strings.HasPrefix(path, m.prefix) {
		// Must be followed by a slash to stay within the subtree
		// e.g., "/users/*" -> prefix "/users"
		return len(path) > len(m.prefix) && path[len(m.prefix)] == '/'
	}
	return false
}

// regexMatcher matches paths using regular expressions.
type regexMatcher struct {
	regex *regexp.Regexp
}

// Match returns true if the path matches the regular expression.
func (m regexMatcher) Match(path string) bool {
	return m.regex.MatchString(path)
}

// NewPolicyRegistry creates a new policy registry.
func NewPolicyRegistry() *PolicyRegistry {
	return &PolicyRegistry{
		rules: make(map[string][]compiledRule),
	}
}

// RequireAuth adds a rule requiring authentication for the specified paths.
func (r *PolicyRegistry) RequireAuth(method string, patterns ...string) *PolicyRegistry {
	for _, pattern := range patterns {
		r.addRule(method, pattern, Rule{RequireAuth: true})
	}
	return r
}

// RequireStepUp adds a rule requiring step-up authentication for the specified paths.
// Step-up authentication implies regular authentication is also required.
func (r *PolicyRegistry) RequireStepUp(method string, patterns ...string) *PolicyRegistry {
	for _, pattern := range patterns {
		r.addRule(method, pattern, Rule{RequireAuth: true, RequireStepUp: true})
	}
	return r
}

// RequireRoles adds a rule requiring specific roles for the specified paths.
func (r *PolicyRegistry) RequireRoles(roles []string, method string, patterns ...string) *PolicyRegistry {
	for _, pattern := range patterns {
		r.addRule(method, pattern, Rule{Roles: roles})
	}
	return r
}

// RequirePermissions adds a rule requiring specific permissions for the specified paths.
func (r *PolicyRegistry) RequirePermissions(perms []string, method string, patterns ...string) *PolicyRegistry {
	for _, pattern := range patterns {
		r.addRule(method, pattern, Rule{Permissions: perms})
	}
	return r
}

// normMethod normalizes the HTTP method to uppercase.
func normMethod(m string) string {
	if m == "" {
		return "*"
	}
	return strings.ToUpper(m)
}

// addRule adds a rule to the registry.
func (r *PolicyRegistry) addRule(method string, pattern string, rule Rule) {
	method = normMethod(method)
	matcher := r.compileMatcher(pattern)
	
	// Merge with existing rule if one exists for this pattern
	existing := r.findRule(method, pattern)
	if existing != nil {
		// Merge rules
		if rule.RequireAuth {
			existing.rule.RequireAuth = true
		}
		if rule.RequireStepUp {
			existing.rule.RequireStepUp = true
		}
		if len(rule.Roles) > 0 {
			existing.rule.Roles = append(existing.rule.Roles, rule.Roles...)
		}
		if len(rule.Permissions) > 0 {
			existing.rule.Permissions = append(existing.rule.Permissions, rule.Permissions...)
		}
	} else {
		// Add new rule
		r.rules[method] = append(r.rules[method], compiledRule{
			pattern: pattern,
			matcher: matcher,
			rule:    rule,
		})
	}
}

// findRule finds an existing rule for a pattern.
func (r *PolicyRegistry) findRule(method string, pattern string) *compiledRule {
	method = normMethod(method)
	for i := range r.rules[method] {
		if r.rules[method][i].pattern == pattern {
			return &r.rules[method][i]
		}
	}
	return nil
}

// compileMatcher creates a path matcher from a pattern.
func (r *PolicyRegistry) compileMatcher(pattern string) pathMatcher {
	// Check if it's a regex (starts and ends with /)
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) > 2 {
		regexStr := pattern[1 : len(pattern)-1]
		regex, err := regexp.Compile(regexStr)
		if err != nil {
			panic(fmt.Sprintf("invalid policy regex %q: %v", pattern, err))
		}
		return regexMatcher{regex: regex}
	}
	
	// Check if it's a wildcard pattern
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return prefixMatcher{prefix: prefix}
	}
	
	// Otherwise, it's an exact match
	return exactMatcher{path: pattern}
}

// GetRules finds all matching rules for a given method and path.
func (r *PolicyRegistry) GetRules(method string, path string) []Rule {
	var matches []Rule
	method = normMethod(method)
	
	// Check method-specific rules
	for _, compiled := range r.rules[method] {
		if compiled.matcher.Match(path) {
			matches = append(matches, compiled.rule)
		}
	}
	
	// Check wildcard method rules
	for _, compiled := range r.rules["*"] {
		if compiled.matcher.Match(path) {
			matches = append(matches, compiled.rule)
		}
	}
	
	return matches
}

// MergeRules combines multiple rules into a single rule.
//
// The most restrictive requirements are kept:
// - RequireAuth: OR (if any requires auth)
// - RequireStepUp: OR (if any requires step-up)
// - Roles: INTERSECT (user must satisfy all matching rules)
// - Permissions: INTERSECT (user must satisfy all matching rules)
func MergeRules(rules []Rule) Rule {
	merged := Rule{}
	var rolesSet map[string]struct{}
	var permsSet map[string]struct{}
	rolesInitialized := false
	permsInitialized := false

	for _, r := range rules {
		if r.RequireAuth {
			merged.RequireAuth = true
		}
		if r.RequireStepUp {
			merged.RequireStepUp = true
		}

		// Roles: intersect if both specify; else carry over specified set
		if len(r.Roles) > 0 {
			if !rolesInitialized {
				rolesSet = toSet(r.Roles)
				rolesInitialized = true
			} else {
				rolesSet = intersect(rolesSet, toSet(r.Roles))
			}
		}
		// Permissions: intersect
		if len(r.Permissions) > 0 {
			if !permsInitialized {
				permsSet = toSet(r.Permissions)
				permsInitialized = true
			} else {
				permsSet = intersect(permsSet, toSet(r.Permissions))
			}
		}
	}

	if rolesInitialized {
		merged.Roles = setKeys(rolesSet)
	}
	if permsInitialized {
		merged.Permissions = setKeys(permsSet)
	}
	return merged
}

// toSet converts a slice of strings to a set.
func toSet(ss []string) map[string]struct{} {
	m := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return m
}

// intersect returns the intersection of two sets.
func intersect(a, b map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{})
	for k := range a {
		if _, ok := b[k]; ok {
			out[k] = struct{}{}
		}
	}
	return out
}

// setKeys returns the keys of a set as a slice.
func setKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
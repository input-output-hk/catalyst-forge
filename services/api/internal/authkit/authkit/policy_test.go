package authkit

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPolicyRegistry_Construction(t *testing.T) {
	registry := NewPolicyRegistry()
	
	assert.NotNil(t, registry, "registry should not be nil")
	assert.NotNil(t, registry.rules, "rules map should be initialized")
	assert.Empty(t, registry.rules, "rules map should be empty on construction")
}

func TestPolicyRegistry_RequireAuth(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Test adding auth requirement
	result := registry.RequireAuth("GET", "/auth/profile", "/auth/settings")
	
	// Should return self for chaining
	assert.Equal(t, registry, result, "RequireAuth should return self for chaining")
	
	// Test rule compilation
	rules := registry.GetRules("GET", "/auth/profile")
	assert.Len(t, rules, 1, "should have one rule for /auth/profile")
	assert.True(t, rules[0].RequireAuth, "should require authentication")
	assert.False(t, rules[0].RequireStepUp, "should not require step-up by default")
	assert.Empty(t, rules[0].Roles, "should not have specific roles")
	assert.Empty(t, rules[0].Permissions, "should not have specific permissions")
}

func TestPolicyRegistry_RequireStepUp(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Test adding step-up requirement
	result := registry.RequireStepUp("POST", "/admin/users", "/admin/roles")
	
	// Should return self for chaining
	assert.Equal(t, registry, result, "RequireStepUp should return self for chaining")
	
	// Test rule compilation
	rules := registry.GetRules("POST", "/admin/users")
	assert.Len(t, rules, 1, "should have one rule for /admin/users")
	assert.True(t, rules[0].RequireAuth, "step-up should imply auth requirement")
	assert.True(t, rules[0].RequireStepUp, "should require step-up")
	assert.Empty(t, rules[0].Roles, "should not have specific roles")
	assert.Empty(t, rules[0].Permissions, "should not have specific permissions")
}

func TestPolicyRegistry_RequireRoles(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Test adding role requirements
	roles := []string{"admin", "moderator"}
	result := registry.RequireRoles(roles, "DELETE", "/api/users/*", "/api/posts/*")
	
	// Should return self for chaining
	assert.Equal(t, registry, result, "RequireRoles should return self for chaining")
	
	// Test rule compilation
	rules := registry.GetRules("DELETE", "/api/users/123")
	assert.Len(t, rules, 1, "should have one rule for /api/users/123")
	assert.False(t, rules[0].RequireAuth, "should not require auth by default")
	assert.False(t, rules[0].RequireStepUp, "should not require step-up by default")
	assert.Equal(t, roles, rules[0].Roles, "should have specified roles")
	assert.Empty(t, rules[0].Permissions, "should not have specific permissions")
}

func TestPolicyRegistry_RequirePermissions(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Test adding permission requirements
	permissions := []string{"read:users", "write:users"}
	result := registry.RequirePermissions(permissions, "PUT", "/api/users/*")
	
	// Should return self for chaining
	assert.Equal(t, registry, result, "RequirePermissions should return self for chaining")
	
	// Test rule compilation
	rules := registry.GetRules("PUT", "/api/users/456")
	assert.Len(t, rules, 1, "should have one rule for /api/users/456")
	assert.False(t, rules[0].RequireAuth, "should not require auth by default")
	assert.False(t, rules[0].RequireStepUp, "should not require step-up by default")
	assert.Empty(t, rules[0].Roles, "should not have specific roles")
	assert.Equal(t, permissions, rules[0].Permissions, "should have specified permissions")
}

func TestPolicyRegistry_MethodNormalization(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Test method normalization
	registry.RequireAuth("get", "/test1")
	registry.RequireAuth("GET", "/test2")
	registry.RequireAuth("", "/test3") // Empty method should become "*"
	
	// Test lowercase method
	rules := registry.GetRules("GET", "/test1")
	assert.Len(t, rules, 1, "lowercase method should be normalized to uppercase")
	
	// Test uppercase method
	rules = registry.GetRules("GET", "/test2")
	assert.Len(t, rules, 1, "uppercase method should work")
	
	// Test wildcard method
	rules = registry.GetRules("POST", "/test3")
	assert.Len(t, rules, 1, "empty method should become wildcard and match any method")
}

func TestExactMatcher(t *testing.T) {
	matcher := exactMatcher{path: "/api/users"}
	
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "exact_match", path: "/api/users", expected: true},
		{name: "different_path", path: "/api/posts", expected: false},
		{name: "prefix_path", path: "/api", expected: false},
		{name: "longer_path", path: "/api/users/123", expected: false},
		{name: "similar_path", path: "/api/user", expected: false},
		{name: "empty_path", path: "", expected: false},
		{name: "root_path", path: "/", expected: false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.path)
			assert.Equal(t, tt.expected, result, "exact matcher for path %q", tt.path)
		})
	}
}

func TestPrefixMatcher(t *testing.T) {
	matcher := prefixMatcher{prefix: "/api/users"}
	
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "exact_match", path: "/api/users", expected: true},
		{name: "with_trailing_slash", path: "/api/users/", expected: true},
		{name: "with_id", path: "/api/users/123", expected: true},
		{name: "with_nested_path", path: "/api/users/123/posts", expected: true},
		{name: "partial_match", path: "/api/user", expected: false},
		{name: "similar_prefix", path: "/api/users-backup", expected: false},
		{name: "shorter_path", path: "/api", expected: false},
		{name: "different_path", path: "/auth/login", expected: false},
		{name: "empty_path", path: "", expected: false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.path)
			assert.Equal(t, tt.expected, result, "prefix matcher for path %q", tt.path)
		})
	}
}

func TestPrefixMatcher_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		path     string
		expected bool
	}{
		{
			name:     "root_prefix",
			prefix:   "/",
			path:     "/anything", 
			expected: false, // "/" doesn't match "/anything" because path[1] must be '/' but it's 'a'
		},
		{
			name:     "empty_prefix",
			prefix:   "",
			path:     "/anything",
			expected: true, // Empty prefix matches everything with HasPrefix
		},
		{
			name:     "prefix_without_leading_slash",
			prefix:   "api",
			path:     "api/users",
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := prefixMatcher{prefix: tt.prefix}
			result := matcher.Match(tt.path)
			assert.Equal(t, tt.expected, result, "prefix %q matching path %q", tt.prefix, tt.path)
		})
	}
}

func TestRegexMatcher(t *testing.T) {
	regex := regexp.MustCompile(`^/api/users/\d+$`)
	matcher := regexMatcher{regex: regex}
	
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "valid_user_id", path: "/api/users/123", expected: true},
		{name: "valid_large_id", path: "/api/users/999999", expected: true},
		{name: "invalid_no_id", path: "/api/users/", expected: false},
		{name: "invalid_non_numeric", path: "/api/users/abc", expected: false},
		{name: "invalid_extra_path", path: "/api/users/123/posts", expected: false},
		{name: "invalid_different_path", path: "/api/posts/123", expected: false},
		{name: "invalid_no_prefix", path: "users/123", expected: false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.path)
			assert.Equal(t, tt.expected, result, "regex matcher for path %q", tt.path)
		})
	}
}

func TestPolicyRegistry_PatternCompilation(t *testing.T) {
	registry := NewPolicyRegistry()
	
	tests := []struct {
		name        string
		pattern     string
		testPath    string
		shouldMatch bool
		description string
	}{
		{
			name:        "exact_pattern",
			pattern:     "/api/health",
			testPath:    "/api/health",
			shouldMatch: true,
			description: "Exact pattern should match exact path",
		},
		{
			name:        "wildcard_pattern",
			pattern:     "/api/users/*",
			testPath:    "/api/users/123",
			shouldMatch: true,
			description: "Wildcard pattern should match subpaths",
		},
		{
			name:        "regex_pattern",
			pattern:     "/^/api/posts/\\d+$/",
			testPath:    "/api/posts/456",
			shouldMatch: true,
			description: "Regex pattern should match valid paths",
		},
		{
			name:        "regex_pattern_no_match",
			pattern:     "/^/api/posts/\\d+$/",
			testPath:    "/api/posts/abc",
			shouldMatch: false,
			description: "Regex pattern should not match invalid paths",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry.RequireAuth("GET", tt.pattern)
			rules := registry.GetRules("GET", tt.testPath)
			
			if tt.shouldMatch {
				assert.Len(t, rules, 1, tt.description)
			} else {
				assert.Empty(t, rules, tt.description)
			}
		})
	}
}

func TestPolicyRegistry_InvalidRegex(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Test that invalid regex patterns panic
	assert.Panics(t, func() {
		registry.RequireAuth("GET", "/[invalid regex/")
	}, "Invalid regex should panic during compilation")
}

func TestPolicyRegistry_RuleMerging(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Add multiple rules for the same pattern
	registry.RequireAuth("GET", "/api/admin")
	registry.RequireStepUp("GET", "/api/admin")
	registry.RequireRoles([]string{"admin"}, "GET", "/api/admin")
	registry.RequirePermissions([]string{"admin:read"}, "GET", "/api/admin")
	
	// Should merge into single rule
	rules := registry.GetRules("GET", "/api/admin")
	assert.Len(t, rules, 1, "should merge into single rule")
	
	rule := rules[0]
	assert.True(t, rule.RequireAuth, "should require auth")
	assert.True(t, rule.RequireStepUp, "should require step-up")
	assert.Contains(t, rule.Roles, "admin", "should contain admin role")
	assert.Contains(t, rule.Permissions, "admin:read", "should contain admin:read permission")
}

func TestPolicyRegistry_MultipleRules(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Add rules for different patterns that might match the same path
	registry.RequireAuth("GET", "/api/*")
	registry.RequireRoles([]string{"user"}, "GET", "/api/profile")
	registry.RequireStepUp("GET", "/^/api/profile$/")
	
	// Test path that matches multiple patterns
	rules := registry.GetRules("GET", "/api/profile")
	assert.Len(t, rules, 3, "should match multiple patterns")
	
	// Verify each rule
	authRule := findRuleWithAuth(rules)
	roleRule := findRuleWithRoles(rules, []string{"user"})
	stepUpRule := findRuleWithStepUp(rules)
	
	assert.NotNil(t, authRule, "should have auth rule")
	assert.NotNil(t, roleRule, "should have role rule")
	assert.NotNil(t, stepUpRule, "should have step-up rule")
}

func TestPolicyRegistry_WildcardMethod(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Add rule for all methods
	registry.RequireAuth("", "/auth/*") // Empty method becomes "*"
	
	// Test different methods
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	for _, method := range methods {
		t.Run("method_"+method, func(t *testing.T) {
			rules := registry.GetRules(method, "/auth/login")
			assert.Len(t, rules, 1, "wildcard method should match %s", method)
			assert.True(t, rules[0].RequireAuth, "should require auth for %s", method)
		})
	}
}

func TestPolicyRegistry_MethodSpecificOverride(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Add general rule and method-specific rule
	registry.RequireAuth("", "/api/data") // All methods
	registry.RequireStepUp("DELETE", "/api/data") // DELETE only
	
	// Test GET (should only have auth requirement)
	getReguls := registry.GetRules("GET", "/api/data")
	assert.Len(t, getReguls, 1, "GET should have one rule")
	assert.True(t, getReguls[0].RequireAuth, "GET should require auth")
	assert.False(t, getReguls[0].RequireStepUp, "GET should not require step-up")
	
	// Test DELETE (should have both auth and step-up)
	deleteRules := registry.GetRules("DELETE", "/api/data")
	assert.Len(t, deleteRules, 2, "DELETE should have two rules")
	
	// Find auth and step-up rules
	authRule := findRuleWithAuth(deleteRules)
	stepUpRule := findRuleWithStepUp(deleteRules)
	
	assert.NotNil(t, authRule, "DELETE should have auth rule")
	assert.NotNil(t, stepUpRule, "DELETE should have step-up rule")
}

func TestMergeRules_EmptyInput(t *testing.T) {
	result := MergeRules([]Rule{})
	
	assert.False(t, result.RequireAuth, "should not require auth")
	assert.False(t, result.RequireStepUp, "should not require step-up")
	assert.Empty(t, result.Roles, "should have no roles")
	assert.Empty(t, result.Permissions, "should have no permissions")
}

func TestMergeRules_SingleRule(t *testing.T) {
	rule := Rule{
		RequireAuth:   true,
		RequireStepUp: true,
		Roles:         []string{"admin", "user"},
		Permissions:   []string{"read", "write"},
	}
	
	result := MergeRules([]Rule{rule})
	
	assert.True(t, result.RequireAuth, "should require auth")
	assert.True(t, result.RequireStepUp, "should require step-up")
	assert.ElementsMatch(t, rule.Roles, result.Roles, "should preserve roles")
	assert.ElementsMatch(t, rule.Permissions, result.Permissions, "should preserve permissions")
}

func TestMergeRules_AuthAndStepUp(t *testing.T) {
	rules := []Rule{
		{RequireAuth: false, RequireStepUp: false},
		{RequireAuth: true, RequireStepUp: false},
		{RequireAuth: false, RequireStepUp: true},
	}
	
	result := MergeRules(rules)
	
	assert.True(t, result.RequireAuth, "should require auth if any rule requires it")
	assert.True(t, result.RequireStepUp, "should require step-up if any rule requires it")
}

func TestMergeRules_RolesIntersection(t *testing.T) {
	rules := []Rule{
		{Roles: []string{"admin", "user", "moderator"}},
		{Roles: []string{"admin", "moderator", "guest"}},
		{Roles: []string{"admin", "superuser"}},
	}
	
	result := MergeRules(rules)
	
	// Intersection should be only "admin"
	assert.Len(t, result.Roles, 1, "should have intersection of roles")
	assert.Contains(t, result.Roles, "admin", "should contain common role")
}

func TestMergeRules_PermissionsIntersection(t *testing.T) {
	rules := []Rule{
		{Permissions: []string{"read", "write", "delete"}},
		{Permissions: []string{"read", "write", "admin"}},
		{Permissions: []string{"read", "execute"}},
	}
	
	result := MergeRules(rules)
	
	// Intersection should be only "read"
	assert.Len(t, result.Permissions, 1, "should have intersection of permissions")
	assert.Contains(t, result.Permissions, "read", "should contain common permission")
}

func TestMergeRules_EmptyIntersection(t *testing.T) {
	rules := []Rule{
		{Roles: []string{"admin"}},
		{Roles: []string{"user"}},
		{Roles: []string{"guest"}},
	}
	
	result := MergeRules(rules)
	
	// No common roles
	assert.Empty(t, result.Roles, "should have empty roles when no intersection")
}

func TestMergeRules_MixedRulesAndEmpty(t *testing.T) {
	rules := []Rule{
		{RequireAuth: true, Roles: []string{"admin", "user"}},
		{RequireStepUp: true}, // No roles/permissions
		{Permissions: []string{"read", "write"}},
	}
	
	result := MergeRules(rules)
	
	assert.True(t, result.RequireAuth, "should require auth")
	assert.True(t, result.RequireStepUp, "should require step-up")
	assert.ElementsMatch(t, []string{"admin", "user"}, result.Roles, "should preserve roles when others don't specify")
	assert.ElementsMatch(t, []string{"read", "write"}, result.Permissions, "should preserve permissions when others don't specify")
}

func TestPolicyRegistry_LargeRuleSet_Performance(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Add many rules to test performance
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for i := 0; i < 1000; i++ {
		for _, method := range methods {
			// Create unique paths for each method and index combination
			registry.RequireAuth(method, fmt.Sprintf("/api/large/%s/%d", method, i))
		}
	}
	
	// Test rule retrieval performance for a specific method and index
	rules := registry.GetRules("GET", "/api/large/GET/500")
	assert.Len(t, rules, 1, "should find rule in large set")
	
	// Test non-matching path
	rules = registry.GetRules("GET", "/api/nonexistent")
	assert.Empty(t, rules, "should not find non-matching rule")
}

func TestPolicyRegistry_ConflictResolution(t *testing.T) {
	registry := NewPolicyRegistry()
	
	// Add conflicting rules for priority testing
	registry.RequireRoles([]string{"admin"}, "GET", "/api/users")
	registry.RequireAuth("GET", "/api/users")
	registry.RequirePermissions([]string{"read:users"}, "GET", "/api/users")
	
	// Should merge into single rule
	rules := registry.GetRules("GET", "/api/users")
	assert.Len(t, rules, 1, "conflicting rules should merge")
	
	rule := rules[0]
	assert.True(t, rule.RequireAuth, "should require auth")
	assert.Contains(t, rule.Roles, "admin", "should contain admin role")
	assert.Contains(t, rule.Permissions, "read:users", "should contain read permission")
}

// Helper functions for tests

func findRuleWithAuth(rules []Rule) *Rule {
	for i := range rules {
		if rules[i].RequireAuth {
			return &rules[i]
		}
	}
	return nil
}

func findRuleWithStepUp(rules []Rule) *Rule {
	for i := range rules {
		if rules[i].RequireStepUp {
			return &rules[i]
		}
	}
	return nil
}

func findRuleWithRoles(rules []Rule, roles []string) *Rule {
	for i := range rules {
		if len(rules[i].Roles) > 0 {
			for _, role := range roles {
				for _, ruleRole := range rules[i].Roles {
					if role == ruleRole {
						return &rules[i]
					}
				}
			}
		}
	}
	return nil
}
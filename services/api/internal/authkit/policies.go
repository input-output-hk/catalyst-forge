package authkit

import libauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"

// BuildPolicies returns a baseline policy registry for new routes.
func BuildPolicies() *libauth.PolicyRegistry {
	reg := libauth.NewPolicyRegistry()
	// Require auth for listing credentials, logout, logout-all
	reg.RequireAuth("GET", "/api/v1/auth/credentials")
	reg.RequireAuth("POST", "/api/v1/auth/logout", "/api/v1/auth/logout-all")
	// Sessions endpoints
	reg.RequireAuth("GET", "/api/v1/auth/sessions")
	reg.RequireAuth("DELETE", "/api/v1/auth/sessions/*")
	// Recovery codes generation
	reg.RequireAuth("POST", "/api/v1/auth/recovery/codes/generate")
	// Step-up for sensitive operations can be added here as needed.
	return reg
}

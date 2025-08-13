package authkit

import libauth "github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"

// BuildPolicies returns a baseline policy registry for new routes.
func BuildPolicies() *libauth.PolicyRegistry {
	reg := libauth.NewPolicyRegistry()
	// Require auth for listing credentials, logout, logout-all
	reg.RequireAuth("GET", "/api/v1/auth/credentials")
	reg.RequireAuth("POST", "/api/v1/auth/logout", "/api/v1/auth/logout-all")
	// Step-up for sensitive operations can be added here as needed.
	return reg
}

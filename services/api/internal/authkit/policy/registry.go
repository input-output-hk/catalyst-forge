package policy

import (
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
)

// BuildRegistry declares route-level policy requirements.
// Applications should extend this to map new endpoints to roles/permissions
// and Step-Up requirements. This function is intentionally conservative and
// should be revised alongside handler/router changes.
func BuildRegistry() *authkit.PolicyRegistry {
	reg := authkit.NewPolicyRegistry()

	// Admin endpoints (example catch-all)
	//reg.RequireRoles([]string{"admin"}, "*", "/admin/*")

	// JWKS must be publicly accessible for clients to validate tokens.
	// We intentionally do NOT add a rule here so the global enforcer
	// will allow it as no-rule → allow.

	// Core domain: releases
	reg.RequirePermissions([]string{"release:read"}, "GET", "/api/v1/releases", "/api/v1/releases/*")
	reg.RequirePermissions([]string{"release:create"}, "POST", "/api/v1/releases")
	reg.RequirePermissions([]string{"release:update"}, "PUT", "/api/v1/releases/*")
	reg.RequirePermissions([]string{"release:delete"}, "DELETE", "/api/v1/releases/*")

	// Core domain: deployments
	reg.RequirePermissions([]string{"deploy:read"}, "GET", "/api/v1/deployments", "/api/v1/deployments/*")
	reg.RequirePermissions([]string{"deploy:create"}, "POST", "/api/v1/deployments")
	reg.RequirePermissions([]string{"deploy:cancel"}, "POST", "/api/v1/deployments/*/cancel")
	reg.RequirePermissions([]string{"deploy:promote"}, "POST", "/api/v1/deployments/*/promote")

	// Projects & environments
	reg.RequirePermissions([]string{"project:read"}, "GET", "/api/v1/projects", "/api/v1/projects/*")
	reg.RequirePermissions([]string{"project:update"}, "PUT", "/api/v1/projects/*")
	reg.RequirePermissions([]string{"env:read"}, "GET", "/api/v1/environments", "/api/v1/environments/*")
	reg.RequirePermissions([]string{"env:update"}, "PUT", "/api/v1/environments/*")

	// Builds / artifacts
	reg.RequirePermissions([]string{"build:trigger"}, "POST", "/api/v1/builds")
	reg.RequirePermissions([]string{"build:read"}, "GET", "/api/v1/builds", "/api/v1/builds/*")
	reg.RequirePermissions([]string{"artifact:write"}, "POST", "/api/v1/artifacts")
	reg.RequirePermissions([]string{"artifact:read"}, "GET", "/api/v1/artifacts", "/api/v1/artifacts/*")

	// PKI: certificate signing
	reg.RequirePermissions([]string{"cert:sign"}, "POST", "/pki/sign")

	// Auth: GitHub OIDC policy management
	// Read
	reg.RequirePermissions([]string{"auth.policies:read"},
		"GET", "/api/v1/auth/oidc/github/policies",
		"GET", "/api/v1/auth/oidc/github/policies/*",
	)
	// Write (mutating endpoints)
	reg.RequirePermissions([]string{"auth.policies:write"},
		"POST", "/api/v1/auth/oidc/github/policies",
		"PUT", "/api/v1/auth/oidc/github/policies/*",
		"DELETE", "/api/v1/auth/oidc/github/policies/*",
	)

	// Auth: user credential management requires authentication
	reg.RequireAuth("GET", "/api/v1/auth/credentials")
	reg.RequireAuth("DELETE", "/api/v1/auth/credentials/*")

	// Sensitive operations may also require step-up in handlers where applicable
	return reg
}

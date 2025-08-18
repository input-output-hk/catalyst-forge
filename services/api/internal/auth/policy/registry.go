package policy

import (
	authperms "github.com/input-output-hk/catalyst-forge/services/api/internal/auth/permissions"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
)

// BuildRegistry declares route-level policy requirements.
func BuildRegistry() *authkit.PolicyRegistry {
	reg := authkit.NewPolicyRegistry()

	// Public endpoints (explicit)
	reg.AllowAnonymous("GET", "/healthz")
	reg.AllowAnonymous("GET", "/.well-known/jwks.json")
	reg.AllowAnonymous("POST", "/api/v1/public/access-requests")

	// Core domain: releases
	reg.RequirePermissions([]string{authperms.ReleaseRead}, "GET", "/api/v1/releases", "/api/v1/releases/*")
	reg.RequirePermissions([]string{authperms.ReleaseCreate}, "POST", "/api/v1/releases")
	reg.RequirePermissions([]string{authperms.ReleaseUpdate}, "PUT", "/api/v1/releases/*")
	reg.RequirePermissions([]string{authperms.ReleaseDelete}, "DELETE", "/api/v1/releases/*")

	// Core domain: deployments
	reg.RequirePermissions([]string{authperms.DeployRead}, "GET", "/api/v1/deployments", "/api/v1/deployments/*")
	reg.RequirePermissions([]string{authperms.DeployCreate}, "POST", "/api/v1/deployments")
	reg.RequirePermissions([]string{authperms.DeployCancel}, "POST", "/api/v1/deployments/*/cancel")
	reg.RequirePermissions([]string{authperms.DeployPromote}, "POST", "/api/v1/deployments/*/promote")

	// Projects & environments
	reg.RequirePermissions([]string{authperms.ProjectRead}, "GET", "/api/v1/projects", "/api/v1/projects/*")
	reg.RequirePermissions([]string{authperms.ProjectUpdate}, "PUT", "/api/v1/projects/*")
	reg.RequirePermissions([]string{authperms.EnvRead}, "GET", "/api/v1/environments", "/api/v1/environments/*")
	reg.RequirePermissions([]string{authperms.EnvUpdate}, "PUT", "/api/v1/environments/*")

	// Builds / artifacts
	reg.RequirePermissions([]string{authperms.BuildTrigger}, "POST", "/api/v1/builds")
	reg.RequirePermissions([]string{authperms.BuildRead}, "GET", "/api/v1/builds", "/api/v1/builds/*")
	reg.RequirePermissions([]string{authperms.ArtifactWrite}, "POST", "/api/v1/artifacts")
	reg.RequirePermissions([]string{authperms.ArtifactRead}, "GET", "/api/v1/artifacts", "/api/v1/artifacts/*")

	// RBAC admin endpoints require rbac:admin
	reg.RequirePermissions([]string{authperms.RBACAdmin}, "*",
		"/api/v1/rbac/*",
		"/api/v1/rbac/roles",
		"/api/v1/rbac/roles/*",
		"/api/v1/rbac/bindings",
		"/api/v1/rbac/bindings/*",
		"/api/v1/rbac/subjects/*",
		"/api/v1/rbac/explain",
		"/api/v1/rbac/conditions",
	)

	// PKI: certificate signing
	reg.RequirePermissions([]string{authperms.CertSign}, "POST", "/pki/sign")

	// Auth: GitHub OIDC policy management
	reg.RequirePermissions([]string{authperms.AuthPoliciesRead},
		"GET", "/api/v1/auth/oidc/github/policies",
		"GET", "/api/v1/auth/oidc/github/policies/*",
	)
	reg.RequirePermissions([]string{authperms.AuthPoliciesWrite},
		"POST", "/api/v1/auth/oidc/github/policies",
		"PUT", "/api/v1/auth/oidc/github/policies/*",
		"DELETE", "/api/v1/auth/oidc/github/policies/*",
	)

	// Auth: user credential management requires authentication
	reg.RequireAuth("GET", "/api/v1/auth/credentials")
	reg.RequireAuth("DELETE", "/api/v1/auth/credentials/*")

	// Access requests: public submit, admin list/decide
	reg.RequirePermissions([]string{authperms.AccessRequestsRead}, "GET", "/api/v1/admin/access-requests")
	reg.RequirePermissions([]string{authperms.AccessRequestsWrite}, "PATCH", "/api/v1/admin/access-requests/*")

	// Auth: profile
	reg.RequireAuth("GET", "/api/v1/auth/me", "/api/v1/auth/session")
	reg.RequireAuth("PATCH", "/api/v1/auth/me")

	// Admin: users listing and management
	reg.RequirePermissions([]string{authperms.UserRead}, "GET", "/api/v1/admin/users")
	reg.RequirePermissions([]string{authperms.UserRead}, "GET", "/api/v1/admin/users/*/credentials")
	reg.RequirePermissions([]string{authperms.UserWrite}, "POST", "/api/v1/admin/users/*/recovery/codes/generate")
	reg.RequirePermissions([]string{authperms.AuditRead}, "GET", "/api/v1/admin/audit")
	reg.RequirePermissions([]string{authperms.UserWrite}, "PATCH", "/api/v1/admin/users/*")
	reg.RequirePermissions([]string{authperms.UserWrite}, "DELETE", "/api/v1/admin/users/*")
	reg.RequirePermissions([]string{authperms.InviteCreate}, "POST", "/api/v1/admin/invites")

	return reg
}

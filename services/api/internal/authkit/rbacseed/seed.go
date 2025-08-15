package rbacseed

import (
	"context"
	"reflect"

	"github.com/google/uuid"
	rbac "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"
)

// Permission keys grouped by domain.
const (
	// Releases
	PermReleaseCreate = rbac.PermissionKey("release:create")
	PermReleaseRead   = rbac.PermissionKey("release:read")
	PermReleaseUpdate = rbac.PermissionKey("release:update")
	PermReleaseDelete = rbac.PermissionKey("release:delete")

	// Deployments
	PermDeployCreate  = rbac.PermissionKey("deploy:create")
	PermDeployRead    = rbac.PermissionKey("deploy:read")
	PermDeployCancel  = rbac.PermissionKey("deploy:cancel")
	PermDeployPromote = rbac.PermissionKey("deploy:promote")

	// Projects
	PermProjectRead   = rbac.PermissionKey("project:read")
	PermProjectUpdate = rbac.PermissionKey("project:update")

	// Environments
	PermEnvRead   = rbac.PermissionKey("env:read")
	PermEnvUpdate = rbac.PermissionKey("env:update")

	// Builds / Artifacts
	PermBuildTrigger  = rbac.PermissionKey("build:trigger")
	PermBuildRead     = rbac.PermissionKey("build:read")
	PermArtifactRead  = rbac.PermissionKey("artifact:read")
	PermArtifactWrite = rbac.PermissionKey("artifact:write")

	// Audit
	PermAuditRead = rbac.PermissionKey("audit:read")

	// Certificates
	PermCertSign = rbac.PermissionKey("cert:sign")

	// Auth policy management
	PermAuthPoliciesRead  = rbac.PermissionKey("auth.policies:read")
	PermAuthPoliciesWrite = rbac.PermissionKey("auth.policies:write")
)

// defaultConditions returns common baseline conditions to apply to entries.
// org_matches is safe in both single- and multi-tenant setups (passes if OrgID missing).
func defaultConditions() []rbac.Condition {
	return []rbac.Condition{{Name: "org_matches"}}
}

// stepUpCondition returns a requires_step_up condition with a 5m window.
func stepUpCondition() rbac.Condition {
	return rbac.Condition{Name: "requires_step_up", Params: map[string]any{"within": "5m"}}
}

// buildBaselineRoles defines the baseline roles without any subject bindings.
func buildBaselineRoles() []rbac.RoleDef {
	// Helper to make entries concise
	allow := func(p rbac.PermissionKey, resType string, conds ...rbac.Condition) rbac.RoleEntry {
		c := append(defaultConditions(), conds...)
		return rbac.RoleEntry{Effect: rbac.Allow, Permission: p, ResourceType: resType, Conditions: c}
	}
	// deny helper reserved for future carve-outs; comment to avoid unused warning
	// deny := func(p rbac.PermissionKey, resType string, conds ...rbac.Condition) rbac.RoleEntry {
	//     c := append(defaultConditions(), conds...)
	//     return rbac.RoleEntry{Effect: rbac.Deny, Permission: p, ResourceType: resType, Conditions: c}
	// }

	maintainer := rbac.RoleDef{
		ID:          uuid.Nil,
		Slug:        "maintainer",
		Name:        "Maintainer",
		Description: "Full project management; step-up on sensitive operations",
		Entries: []rbac.RoleEntry{
			// Releases (project-scoped)
			allow(PermReleaseCreate, "project"),
			allow(PermReleaseRead, "project"),
			allow(PermReleaseUpdate, "project"),
			allow(PermReleaseDelete, "project", stepUpCondition()),
			// Deployments
			allow(PermDeployCreate, "project"),
			allow(PermDeployRead, "project"),
			allow(PermDeployCancel, "project"),
			allow(PermDeployPromote, "project", stepUpCondition()),
			// Projects / Environments
			allow(PermProjectRead, "project"),
			allow(PermProjectUpdate, "project", stepUpCondition()),
			allow(PermEnvRead, "project"),
			allow(PermEnvUpdate, "project", stepUpCondition()),
			// Builds / Artifacts
			allow(PermBuildTrigger, "project"),
			allow(PermBuildRead, "project"),
			allow(PermArtifactRead, "project"),
			allow(PermArtifactWrite, "project"),
			// Auth policy management (global scope)
			allow(PermAuthPoliciesRead, "global"),
			allow(PermAuthPoliciesWrite, "global", stepUpCondition()),
			// Audit
			allow(PermAuditRead, "project"),
			// Certificates: unrestricted by default for maintainers (service-level RBAC conditions can still restrict)
			allow(PermCertSign, "global"),
			// Example deny carve-out placeholder (none by default)
			// _ = deny // keep compiler happy if deny unused
		},
		Version: 1,
	}

	developer := rbac.RoleDef{
		Slug:        "developer",
		Name:        "Developer",
		Description: "Developers: draft releases, trigger builds, limited deploys",
		Entries: []rbac.RoleEntry{
			allow(PermReleaseCreate, "project"),
			allow(PermReleaseRead, "project"),
			allow(PermReleaseUpdate, "project"),
			// No delete
			allow(PermDeployCreate, "project"),
			allow(PermDeployRead, "project"),
			// No promote/cancel by default
			allow(PermProjectRead, "project"),
			allow(PermEnvRead, "project"),
			allow(PermBuildTrigger, "project"),
			allow(PermBuildRead, "project"),
			allow(PermArtifactRead, "project"),
			allow(PermArtifactWrite, "project"),
			// Certificates: not granted by default
		},
		Version: 1,
	}

	qa := rbac.RoleDef{
		Slug:        "qa",
		Name:        "QA",
		Description: "Quality assurance: read and promote in controlled contexts",
		Entries: []rbac.RoleEntry{
			allow(PermReleaseRead, "project"),
			allow(PermDeployRead, "project"),
			// Allow promote with step-up; environment-specific gating can be added via additional conditions later
			allow(PermDeployPromote, "project", stepUpCondition()),
			allow(PermProjectRead, "project"),
			allow(PermEnvRead, "project"),
		},
		Version: 1,
	}

	viewer := rbac.RoleDef{
		Slug:        "viewer",
		Name:        "Viewer",
		Description: "Read-only access across project resources",
		Entries: []rbac.RoleEntry{
			allow(PermReleaseRead, "project"),
			allow(PermDeployRead, "project"),
			allow(PermProjectRead, "project"),
			allow(PermEnvRead, "project"),
			allow(PermBuildRead, "project"),
			allow(PermArtifactRead, "project"),
			allow(PermAuditRead, "project"),
		},
		Version: 1,
	}

	ciRunner := rbac.RoleDef{
		Slug:        "ci_runner",
		Name:        "CI Runner",
		Description: "CI pipelines: trigger builds and publish artifacts",
		Entries: []rbac.RoleEntry{
			allow(PermBuildTrigger, "project"),
			allow(PermBuildRead, "project"),
			allow(PermArtifactWrite, "project"),
			allow(PermArtifactRead, "project"),
		},
		Version: 1,
	}

	return []rbac.RoleDef{maintainer, developer, qa, viewer, ciRunner}
}

// SeedDefaultRoles creates or updates baseline roles.
// If bump is true, role version is incremented when entries change to invalidate caches.
func SeedDefaultRoles(ctx context.Context, s rbac.Store, bump bool) error {
	roles := buildBaselineRoles()
	for _, role := range roles {
		if err := ensureRole(ctx, s, role, bump); err != nil {
			return err
		}
	}
	return nil
}

func ensureRole(ctx context.Context, s rbac.Store, desired rbac.RoleDef, bump bool) error {
	existing, err := s.GetRole(ctx, desired.Slug)
	if err != nil {
		return err
	}
	if existing == nil {
		// Create new role with desired version (default 1)
		if desired.ID == uuid.Nil {
			desired.ID = uuid.New()
		}
		return s.CreateRole(ctx, desired)
	}

	// Compare entries and basic fields (ignore ID and Version here)
	same := existing.Name == desired.Name && existing.Description == desired.Description && reflect.DeepEqual(existing.Entries, desired.Entries)
	if same {
		return nil
	}

	// Update entries; bump version if requested
	newVersion := existing.Version
	if bump {
		newVersion = existing.Version + 1
		if err := s.BumpRoleVersion(ctx, existing.Slug); err != nil {
			// fall back to setting version via UpdateRole
			newVersion = existing.Version + 1
		}
	}
	desired.ID = existing.ID
	desired.Version = newVersion
	return s.UpdateRole(ctx, desired)
}

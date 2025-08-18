package auth

import (
	"context"
	"reflect"

	"github.com/google/uuid"
	authperms "github.com/input-output-hk/catalyst-forge/services/api/internal/auth/permissions"
	rbac "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"
)

// defaultConditions returns common baseline conditions to apply to entries.
func defaultConditions() []rbac.Condition { return []rbac.Condition{{Name: "org_matches"}} }

// stepUpCondition returns a requires_step_up condition with a 5m window.
func stepUpCondition() rbac.Condition {
	return rbac.Condition{Name: "requires_step_up", Params: map[string]any{"within": "5m"}}
}

// buildBaselineRoles defines the baseline roles without any subject bindings.
func buildBaselineRoles() []rbac.RoleDef {
	allow := func(p rbac.PermissionKey, resType string, conds ...rbac.Condition) rbac.RoleEntry {
		c := append(defaultConditions(), conds...)
		return rbac.RoleEntry{Effect: rbac.Allow, Permission: p, ResourceType: resType, Conditions: c}
	}

	maintainer := rbac.RoleDef{
		ID:          uuid.Nil,
		Slug:        "maintainer",
		Name:        "Maintainer",
		Description: "Full project management; step-up on sensitive operations",
		Color:       "#1d4ed8",
		Entries: []rbac.RoleEntry{
			allow(rbac.PermissionKey(authperms.ReleaseCreate), "project"),
			allow(rbac.PermissionKey(authperms.ReleaseRead), "project"),
			allow(rbac.PermissionKey(authperms.ReleaseUpdate), "project"),
			allow(rbac.PermissionKey(authperms.ReleaseDelete), "project", stepUpCondition()),
			allow(rbac.PermissionKey(authperms.DeployCreate), "project"),
			allow(rbac.PermissionKey(authperms.DeployRead), "project"),
			allow(rbac.PermissionKey(authperms.DeployCancel), "project"),
			allow(rbac.PermissionKey(authperms.DeployPromote), "project", stepUpCondition()),
			allow(rbac.PermissionKey(authperms.ProjectRead), "project"),
			allow(rbac.PermissionKey(authperms.ProjectUpdate), "project", stepUpCondition()),
			allow(rbac.PermissionKey(authperms.EnvRead), "project"),
			allow(rbac.PermissionKey(authperms.EnvUpdate), "project", stepUpCondition()),
			allow(rbac.PermissionKey(authperms.BuildTrigger), "project"),
			allow(rbac.PermissionKey(authperms.BuildRead), "project"),
			allow(rbac.PermissionKey(authperms.ArtifactRead), "project"),
			allow(rbac.PermissionKey(authperms.ArtifactWrite), "project"),
			allow(rbac.PermissionKey(authperms.AuthPoliciesRead), "global"),
			allow(rbac.PermissionKey(authperms.AuthPoliciesWrite), "global", stepUpCondition()),
			allow(rbac.PermissionKey(authperms.AuditRead), "project"),
			allow(rbac.PermissionKey(authperms.CertSign), "global"),
		},
		Version: 1,
	}

	developer := rbac.RoleDef{
		Slug:        "developer",
		Name:        "Developer",
		Description: "Developers: draft releases, trigger builds, limited deploys",
		Color:       "#059669",
		Entries: []rbac.RoleEntry{
			allow(rbac.PermissionKey(authperms.ReleaseCreate), "project"),
			allow(rbac.PermissionKey(authperms.ReleaseRead), "project"),
			allow(rbac.PermissionKey(authperms.ReleaseUpdate), "project"),
			allow(rbac.PermissionKey(authperms.DeployCreate), "project"),
			allow(rbac.PermissionKey(authperms.DeployRead), "project"),
			allow(rbac.PermissionKey(authperms.ProjectRead), "project"),
			allow(rbac.PermissionKey(authperms.EnvRead), "project"),
			allow(rbac.PermissionKey(authperms.BuildTrigger), "project"),
			allow(rbac.PermissionKey(authperms.BuildRead), "project"),
			allow(rbac.PermissionKey(authperms.ArtifactRead), "project"),
			allow(rbac.PermissionKey(authperms.ArtifactWrite), "project"),
		},
		Version: 1,
	}

	qa := rbac.RoleDef{
		Slug:        "qa",
		Name:        "QA",
		Description: "Quality assurance: read and promote in controlled contexts",
		Color:       "#7c3aed",
		Entries: []rbac.RoleEntry{
			allow(rbac.PermissionKey(authperms.ReleaseRead), "project"),
			allow(rbac.PermissionKey(authperms.DeployRead), "project"),
			allow(rbac.PermissionKey(authperms.DeployPromote), "project", stepUpCondition()),
			allow(rbac.PermissionKey(authperms.ProjectRead), "project"),
			allow(rbac.PermissionKey(authperms.EnvRead), "project"),
		},
		Version: 1,
	}

	viewer := rbac.RoleDef{
		Slug:        "viewer",
		Name:        "Viewer",
		Description: "Read-only access across project resources",
		Color:       "#6b7280",
		Entries: []rbac.RoleEntry{
			allow(rbac.PermissionKey(authperms.ReleaseRead), "project"),
			allow(rbac.PermissionKey(authperms.DeployRead), "project"),
			allow(rbac.PermissionKey(authperms.ProjectRead), "project"),
			allow(rbac.PermissionKey(authperms.EnvRead), "project"),
			allow(rbac.PermissionKey(authperms.BuildRead), "project"),
			allow(rbac.PermissionKey(authperms.ArtifactRead), "project"),
			allow(rbac.PermissionKey(authperms.AuditRead), "project"),
		},
		Version: 1,
	}

	ciRunner := rbac.RoleDef{
		Slug:        "ci_runner",
		Name:        "CI Runner",
		Description: "CI pipelines: trigger builds and publish artifacts",
		Color:       "#d97706",
		Entries: []rbac.RoleEntry{
			allow(rbac.PermissionKey(authperms.BuildTrigger), "project"),
			allow(rbac.PermissionKey(authperms.BuildRead), "project"),
			allow(rbac.PermissionKey(authperms.ArtifactWrite), "project"),
			allow(rbac.PermissionKey(authperms.ArtifactRead), "project"),
		},
		Version: 1,
	}

	return []rbac.RoleDef{maintainer, developer, qa, viewer, ciRunner}
}

// AdminRoleDef returns the administrator role with broad management permissions.
func AdminRoleDef() rbac.RoleDef {
	allow := func(p rbac.PermissionKey, resType string, conds ...rbac.Condition) rbac.RoleEntry {
		return rbac.RoleEntry{Effect: rbac.Allow, Permission: p, ResourceType: resType, Conditions: conds}
	}
	entries := make([]rbac.RoleEntry, 0, len(authperms.All()))
	for _, p := range authperms.All() {
		entries = append(entries, allow(rbac.PermissionKey(p), ""))
	}
	return rbac.RoleDef{Slug: "admin", Name: "Administrator", Description: "Platform administrators with full management permissions", Color: "#b91c1c", Entries: entries, Version: 1}
}

// SeedDefaultRoles creates or updates baseline roles.
func SeedDefaultRoles(ctx context.Context, s rbac.Store, bump bool) error {
	roles := buildBaselineRoles()
	for _, role := range roles {
		if err := ensureRole(ctx, s, role, bump); err != nil {
			return err
		}
	}
	return nil
}

// EnsureAdminRole seeds or updates the admin role independently.
func EnsureAdminRole(ctx context.Context, s rbac.Store) error {
	desired := AdminRoleDef()
	return ensureRole(ctx, s, desired, true)
}

func ensureRole(ctx context.Context, s rbac.Store, desired rbac.RoleDef, bump bool) error {
	existing, err := s.GetRole(ctx, desired.Slug)
	if err != nil {
		return err
	}
	if existing == nil {
		if desired.ID == uuid.Nil {
			desired.ID = uuid.New()
		}
		return s.CreateRole(ctx, desired)
	}
	same := existing.Name == desired.Name && existing.Description == desired.Description && reflect.DeepEqual(existing.Entries, desired.Entries) && existing.Color == desired.Color
	if same {
		return nil
	}
	newVersion := existing.Version
	if bump {
		newVersion = existing.Version + 1
		if err := s.BumpRoleVersion(ctx, existing.Slug); err != nil {
			newVersion = existing.Version + 1
		}
	}
	desired.ID = existing.ID
	desired.Version = newVersion
	return s.UpdateRole(ctx, desired)
}

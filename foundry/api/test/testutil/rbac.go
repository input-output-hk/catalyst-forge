//go:build integration

package testutil

import (
	"context"
	"fmt"

	rbac "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/rbac"
	gstor "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/rbac/gormstore"
	"gorm.io/gorm"
)

// SeedCertRole creates a role with cert:sign and a dns_sans_suffix_in condition.
func SeedCertRole(ctx context.Context, db *gorm.DB, roleSlug string, allowedSuffixes []string) error {
	s := gstor.New(db)
	// Build conditions
	params := make([]any, 0, len(allowedSuffixes))
	for _, suf := range allowedSuffixes {
		params = append(params, suf)
	}
	cond := rbac.Condition{Name: "dns_sans_suffix_in", Params: map[string]any{"suffixes": params}}
	role := rbac.RoleDef{Slug: roleSlug, Name: roleSlug, Entries: []rbac.RoleEntry{{Effect: rbac.Allow, Permission: rbac.PermissionKey("cert:sign"), ResourceType: "cert-request", Conditions: []rbac.Condition{cond}}}}
	if err := s.CreateRole(ctx, role); err != nil {
		// Try update if exists
		if err2 := s.UpdateRole(ctx, role); err2 != nil {
			return fmt.Errorf("rbac seed role: %w / %v", err, err2)
		}
	}
	return nil
}

// BindRole binds a role to a subject at global scope.
func BindRole(ctx context.Context, db *gorm.DB, subjectID string, roleSlug string) error {
	s := gstor.New(db)
	b := rbac.Binding{Subject: rbac.Subject{Type: rbac.SubjectUser, ID: subjectID}, RoleSlug: roleSlug, ScopeType: rbac.ScopeGlobal, ScopeID: ""}
	return s.AddBinding(ctx, b)
}

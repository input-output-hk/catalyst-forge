//go:build gormsqlite

package gormstore

import (
	"context"
	"testing"

	r "github.com/catalystgo/catalyst-forge/lib/foundry/authkit/rbac"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Role{}, &RoleEntry{}, &Binding{}, &PrincipalVersion{}))
	return db
}

func TestStore_RolesCRUD(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	s := New(db)

	// Create role
	role := r.RoleDef{Slug: "ops", Name: "Operators", Entries: []r.RoleEntry{{Effect: r.Allow, Permission: "infra:deploy", ResourceType: "project"}}}
	require.NoError(t, s.CreateRole(context.Background(), role))

	// Get role
	got, err := s.GetRole(context.Background(), "ops")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "ops", got.Slug)
	assert.Len(t, got.Entries, 1)

	// Update role
	got.Entries = []r.RoleEntry{{Effect: r.Deny, Permission: "infra:deploy", ResourceType: "project"}}
	require.NoError(t, s.UpdateRole(context.Background(), *got))
	got2, err := s.GetRole(context.Background(), "ops")
	require.NoError(t, err)
	require.NotNil(t, got2)
	require.Len(t, got2.Entries, 1)
	assert.Equal(t, r.Deny, got2.Entries[0].Effect)

	// List roles
	list, err := s.ListRoles(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestStore_BindingsAndPrincipalVersion(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	s := New(db)

	// Seed role
	require.NoError(t, s.CreateRole(context.Background(), r.RoleDef{Slug: "reader", Entries: []r.RoleEntry{{Effect: r.Allow, Permission: "data:read", ResourceType: "project"}}}))

	subj := r.Subject{Type: r.SubjectUser, ID: uuid.New().String()}
	// Initially version zero
	v0, err := s.GetPrincipalVersion(context.Background(), subj)
	require.NoError(t, err)
	assert.Equal(t, int64(0), v0)

	// Add binding
	b := r.Binding{ID: uuid.New(), Subject: subj, RoleSlug: "reader", ScopeType: r.ScopeProject, ScopeID: "p1"}
	require.NoError(t, s.AddBinding(context.Background(), b))

	// List bindings
	bs, err := s.ListBindings(context.Background(), subj)
	require.NoError(t, err)
	require.Len(t, bs, 1)

	// Bump principal version
	require.NoError(t, s.BumpPrincipalVersion(context.Background(), subj))
	v1, err := s.GetPrincipalVersion(context.Background(), subj)
	require.NoError(t, err)
	assert.Equal(t, int64(1), v1)

	// Remove binding
	require.NoError(t, s.RemoveBinding(context.Background(), b.ID))
}

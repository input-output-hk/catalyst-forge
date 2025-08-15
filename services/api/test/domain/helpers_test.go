//go:build integration

package domain

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

func newDomainEnv(t *testing.T) *tu.Env {
	t.Helper()
	if suite == nil {
		t.Fatalf("suite not initialized")
	}
	env, err := tu.NewTestEnv(t.Context(), suite)
	if err != nil {
		t.Fatalf("failed to create env: %v", err)
	}
	t.Cleanup(env.Close)
	return env
}

func withBypassHeaders(h map[string]string) map[string]string {
	if h == nil {
		h = map[string]string{}
	}
	// Enable test auth bypass by providing headers + env
	h["X-Test-User-ID"] = "00000000-0000-0000-0000-000000000000"
	h["X-Test-Email"] = "test@foundry.dev"
	// grant read permissions by default; tests may override to include write
	h["X-Test-Roles"] = "user"
	h["X-Test-Permissions"] = "release:read,project:read,build:read,artifact:read,env:read"
	return h
}

func authHeaders(env *tu.Env) map[string]string {
	return map[string]string{"Authorization": "Bearer " + env.AdminJWT}
}

func openGormFromSuite(t *testing.T) *gorm.DB {
	to := t
	to.Helper()
	if suite == nil {
		to.Fatalf("suite not initialized")
	}
	host, port, user, pass, dbname, ssl, err := suite.PG.DSN(t.Context())
	if err != nil {
		to.Fatalf("dsn: %v", err)
	}
	dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + pass + " dbname=" + dbname + " sslmode=" + ssl
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		to.Fatalf("open gorm: %v", err)
	}
	return db
}

// seedRepoProject inserts a minimal repository and project and returns their IDs.
func seedRepoProject(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()
	db := openGormFromSuite(t)
	sqlDB, err := db.DB()
	if err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}

	repoID := uuid.New()
	projID := uuid.New()
	// Insert repository
	if err := db.Exec(
		"INSERT INTO repository (id, host, org, name, default_branch, created_at, updated_at) VALUES (?, ?, ?, ?, 'main', now(), now())",
		repoID, "github.com", "acme", "demo",
	).Error; err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	// Insert project (status defaults to active)
	if err := db.Exec(
		"INSERT INTO project (id, repo_id, path, slug, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'active', now(), now())",
		projID, repoID, "app", "app",
	).Error; err != nil {
		t.Fatalf("insert project: %v", err)
	}
	return repoID, projID
}

//go:build integration

package certs

import (
	"context"
	"fmt"
	"testing"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newCertsEnv provides a per-test environment backed by the package-level suite from TestMain.
func newCertsEnv(t *testing.T) *tu.Env {
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

// openCertsGorm opens a gorm DB using the suite's Postgres container connection info.
func openCertsGorm(t *testing.T) *gorm.DB {
	t.Helper()
	if suite == nil || suite.PG == nil {
		t.Fatalf("suite not initialized")
	}
	host, port, user, pass, db, ssl, err := suite.PG.DSN(context.Background())
	if err != nil {
		t.Fatalf("pg dsn: %v", err)
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", host, port, user, pass, db, ssl)
	gdb, gerr := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if gerr != nil {
		t.Fatalf("open gorm: %v", gerr)
	}
	return gdb
}

func closeGorm(t *testing.T, db *gorm.DB) {
	t.Helper()
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

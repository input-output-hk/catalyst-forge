//go:build integration

package test

import (
	"context"
	"testing"

	"github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// NewTestEnv creates a fresh test environment for a test.
func NewTestEnv(t *testing.T) *testutil.Env {
	t.Helper()
	if suite == nil {
		t.Fatalf("suite not initialized")
	}

	email := "admin-" + testutil.RandomHex(6) + "@foundry.dev"
	t.Logf("Starting fresh env for %s", email)

	env, err := suite.PerTestEnv(context.Background(), email)
	if err != nil {
		t.Fatalf("failed to create per-test env: %v", err)
	}

	t.Logf("Fresh env up at %s", env.BaseURL())
	t.Cleanup(env.Close)

	return env
}

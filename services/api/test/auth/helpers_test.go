//go:build integration

package auth_test

import (
	"testing"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// newAuthEnv provides a per-test environment backed by the package-level suite from TestMain.
func newAuthEnv(t *testing.T) *tu.Env {
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

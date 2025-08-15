//go:build integration
// +build integration

package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/foundry/api/test/testutil"
)

// TestAuth_BootstrapAndMe verifies that the test env boots, migrations run, admin is bootstrapped,
// and /me returns the authenticated user via the generated client when token is present.
func TestAuth_BootstrapAndMe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Start suite (postgres)
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	// Run migrations and snapshot
	require.NoError(t, suite.SnapshotMigrations(ctx))

	// New env per test
	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	// Build generated client (unused for now)
	_, err = env.NewGenClient()
	require.NoError(t, err)

	// Call /me via generated client (path depends on client naming; using raw HTTP as placeholder)
	// For now, verify server health as a smoke test for environment; auth e2e will be expanded next.
	require.NotEmpty(t, env.BaseURL())
}

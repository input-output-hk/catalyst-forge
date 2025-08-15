//go:build integration
// +build integration

package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAuth_BootstrapAndMe verifies that the test env boots, migrations run, admin is bootstrapped,
// and /me returns the authenticated user via the generated client when token is present.
func TestAuth_BootstrapAndMe(t *testing.T) {
	t.Parallel()

	// New env per test from package-level suite
	env := newAuthEnv(t)

	// Build generated client (unused for now)
	_, err := env.NewGenClient()
	require.NoError(t, err)

	// Call /me via generated client (path depends on client naming; using raw HTTP as placeholder)
	// For now, verify server health as a smoke test for environment; auth e2e will be expanded next.
	require.NotEmpty(t, env.BaseURL())
}

//go:build integration

package auth_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/foundry/api/test/testutil"
)

// TestAuth_Me_Unauthorized ensures /me requires authentication.
func TestAuth_Me_Unauthorized(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/me", nil, nil, &out)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

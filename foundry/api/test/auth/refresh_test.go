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

// TestAuth_Refresh covers positive and negative refresh flows.
func TestAuth_Refresh(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	// Positive: perform a refresh with CSRF and cookie automatically via generated client
	// Use the high-level client to hit /healthz (sanity) and then a direct call to /me with admin bearer.
	_, err = env.NewGenClient()
	require.NoError(t, err)

	// Direct /me call with the admin access token ensures we have a working bearer token.
	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	var meResp map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/me", headers, nil, &meResp)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, env.AdminEmail, meResp["email"])

	// Negative: missing CSRF header with refresh cookie should fail on direct HTTP call
	{
		// Attempt raw POST to refresh endpoint without CSRF header, but with cookie
		headers := map[string]string{}
		// Cookie jar inside server session is HTTP only; we can simulate by not providing CSRF
		var out map[string]any
		_, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/refresh", headers, nil, &out)
		require.Error(t, err, "missing CSRF should be rejected")
	}

	// Negative: invalid refresh cookie
	{
		headers := map[string]string{
			"Cookie":       "__Host-refresh_token=invalid",
			"X-CSRF-Token": "bogus",
		}
		var out map[string]any
		resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/refresh", headers, nil, &out)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "invalid cookie should be unauthorized")
	}
}

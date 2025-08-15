//go:build integration

package auth_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/foundry/api/test/testutil"
)

func TestAuth_Me_AfterBootstrap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	// Call /me directly using http helper with admin token if provided
	headers := map[string]string{}
	if env.AdminJWT != "" {
		headers["Authorization"] = "Bearer " + env.AdminJWT
	}
	var meResp map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/me", headers, nil, &meResp)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, env.AdminEmail, meResp["email"]) // best-effort check
}

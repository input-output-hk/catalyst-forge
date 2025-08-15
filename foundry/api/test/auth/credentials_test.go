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

// TestAuth_Credentials_List_And_Delete exercises listing and deletion auth requirements.
func TestAuth_Credentials_List_And_Delete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	require.NotEmpty(t, env.AdminJWT)

	// List requires auth
	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	var list map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/credentials", headers, nil, &list)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Delete unauthorized (no auth header). Expect 401.
	var out map[string]any
	resp, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/auth/credentials/1234", nil, nil, &out)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

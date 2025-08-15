//go:build integration

package auth_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/foundry/api/test/testutil"
)

// TestAuth_Recovery_InitAndVerify exercises the recovery init and verify endpoints (happy path + negative).
func TestAuth_Recovery_InitAndVerify(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	// Recovery init
	var initOut map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/recovery/init", nil, map[string]string{
		"email": env.AdminEmail,
	}, &initOut)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	flowID, _ := initOut["flow_id"].(string)
	require.NotEmpty(t, flowID)

	// Negative verify: wrong code
	var verifyOut map[string]any
	_, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/recovery/verify", nil, map[string]string{
		"flow_id": flowID,
		"code":    "bad-code",
	}, &verifyOut)
	require.Error(t, err)
}

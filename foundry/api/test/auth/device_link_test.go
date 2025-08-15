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

// TestAuth_DeviceLink_Begin_And_Exchange_Pending exercises begin and exchange pending path.
func TestAuth_DeviceLink_Begin_And_Exchange_Pending(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	// Begin
	var begin map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/device-link/begin", nil, map[string]any{
		"device_name": "it",
		"purpose":     "login",
	}, &begin)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	deviceCode, _ := begin["device_code"].(string)
	require.NotEmpty(t, deviceCode)

	// Immediate exchange should return authorization_pending
	var exch map[string]any
	resp, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/device-link/exchange", nil, map[string]any{
		"device_code": deviceCode,
	}, &exch)
	// helper returns error on non-2xx; tolerate and assert status
	if err != nil {
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	}
	assert.Equal(t, "authorization_pending", exch["error"])
}

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

// TestAuth_DeviceLink_Verify_Negative ensures verify returns 400 for missing/invalid code.
func TestAuth_DeviceLink_Verify_Negative(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	// Missing code param
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/device-link/verify", nil, nil, nil)
	require.Error(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	}

	// Invalid code param
	resp, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/device-link/verify?code=NOPE-12", nil, nil, nil)
	require.Error(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	}
}

// TestAuth_DeviceLink_Authorize_Requires_StepUp ensures 428 when attempting authorize without fresh step-up.
func TestAuth_DeviceLink_Authorize_Requires_StepUp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	// Begin to get a user_code
	var begin map[string]any
	_, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/device-link/begin", nil, map[string]any{
		"device_name": "it",
		"purpose":     "login",
	}, &begin)
	require.NoError(t, err)

	userCode, _ := begin["user_code"].(string)
	require.NotEmpty(t, userCode)

	// Attempt authorize with admin bearer but without step-up -> expect 428
	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/device-link/authorize", headers, map[string]any{
		"user_code": userCode,
	}, &out)
	require.Error(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusPreconditionRequired, resp.StatusCode)
	}
}

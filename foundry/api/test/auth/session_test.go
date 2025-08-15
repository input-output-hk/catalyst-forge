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

// TestAuth_Session covers the session endpoint with and without auth.
func TestAuth_Session(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	// Positive: with bearer
	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	var sess map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/session", headers, nil, &sess)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, true, sess["valid"])
	// Expect step-up required initially (no step-up performed yet)
	if v, ok := sess["step_up_required"].(bool); ok {
		assert.Equal(t, true, v)
	}

	// Negative: no bearer
	var out map[string]any
	resp, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/session", nil, nil, &out)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

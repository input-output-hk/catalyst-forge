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

// TestAuth_Credentials_Add_Negatives covers add begin/complete negative cases without valid ceremony.
func TestAuth_Credentials_Add_Negatives(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}

	// Begin add should accept request; in headless test we expect 400 on malformed body
	var begin map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/credentials/add/begin", headers, map[string]any{
		"display_name": "test-key",
	}, &begin)
	// Handler may return 200 with a challenge or 400 if inputs invalid; both acceptable for this negative smoke
	require.NoError(t, err)
	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, resp.StatusCode)

	// Complete without proper challenge should fail
	var complete map[string]any
	resp, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/credentials/add/complete", headers, map[string]any{
		"credential": map[string]any{"id": "bad", "rawId": "bad", "response": map[string]any{}},
	}, &complete)
	require.Error(t, err)
	if resp != nil {
		assert.Contains(t, []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden}, resp.StatusCode)
	}
}

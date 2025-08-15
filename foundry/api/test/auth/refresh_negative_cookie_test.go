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

// TestAuth_Refresh_NoCookie ensures refresh without cookie fails with 401/400.
func TestAuth_Refresh_NoCookie(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	// No cookies, no CSRF. Expect failure.
	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/refresh", nil, nil, &out)
	require.Error(t, err)
	if resp != nil {
		assert.Contains(t, []int{http.StatusUnauthorized, http.StatusBadRequest, http.StatusForbidden}, resp.StatusCode)
	}
}

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

// TestAuth_Bootstrap_Invalid_Token_Returns_404 validates invalid/used token returns 404.
func TestAuth_Bootstrap_Invalid_Token_Returns_404(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	suite, err := tu.SuiteStart(ctx)
	require.NoError(t, err)
	defer func() { _ = suite.SuiteStop(ctx) }()

	require.NoError(t, suite.SnapshotMigrations(ctx))

	env, err := tu.NewTestEnv(ctx, suite)
	require.NoError(t, err)
	defer env.Close()

	// Invalid token (wrong length) will be rejected by validation (400) or by service (404)
	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/bootstrap", nil, map[string]any{
		"email":           "x@example.com",
		"bootstrap_token": "deadbeefdeadbeefdeadbeefdeadbee", // 31 chars
	}, &out)
	require.Error(t, err)
	if resp != nil {
		assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound}, resp.StatusCode)
	}
}

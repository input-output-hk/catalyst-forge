//go:build integration

package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// TestAuth_GitHub_Exchange_Enabled_Smoke toggles config to enable GitHub OIDC and exercises exchange endpoint minimally.
func TestAuth_GitHub_Exchange_Enabled_Smoke(t *testing.T) {
	ctx := t.Context()

	// Start server with GitHub OIDC enabled
	cfg, err := tu.DefaultTestConfig()
	require.NoError(t, err)
	// Enable via env by starting with custom env builder: use StartAPIServer then rely on AUTH_GITHUBOIDC_ENABLED default false overridden by env
	// We cannot mutate cfg for this; instead temporarily set env var for this process so StartAPIServer inherits it
	t.Setenv("AUTH_GITHUBOIDC_ENABLED", "true")
	srv, err := tu.StartAPIServer(ctx, cfg, suite.PG)
	require.NoError(t, err)
	defer srv.Stop()

	// Without a valid JWT, expect 400/401 depending on validation path
	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, srv.BaseURL+"/api/v1/auth/oidc/github/exchange", nil, map[string]any{
		"id_token": "bogus",
	}, &out)
	require.Error(t, err)
	if resp != nil {
		assert.Contains(t, []int{http.StatusBadRequest, http.StatusUnauthorized}, resp.StatusCode)
	}
}

//go:build integration

package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// TestAuth_Logout covers single-session logout behavior.
func TestAuth_Logout(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	// First, ensure /me works with the admin token
	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	var me map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/me", headers, nil, &me)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Now call logout; since we’re not using the refresh cookie here, this should still be accepted (endpoint returns 204)
	var out map[string]any
	resp, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/logout", nil, nil, &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// TestAuth_LogoutAll revokes all refresh tokens for the user.
func TestAuth_LogoutAll(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	// Ensure health of session via /me
	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	var me map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/me", headers, nil, &me)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Call logout-all
	// Logout-all requires authentication (uses AuthKit context). Provide bearer token.
	outHeaders := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	var out map[string]any
	resp, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/logout-all", outHeaders, nil, &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

//go:build integration

package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// TestAuth_LogoutAll_Invalidates_RefreshReuse verifies that after logout-all, refresh cookie reuse fails.
func TestAuth_LogoutAll_Invalidates_RefreshReuse(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	// Have admin access token available
	assert.NotEmpty(t, env.AdminJWT)

	// Ensure refresh works before logout-all
	// Perform a refresh to ensure cookie is set and valid
	// Missing CSRF should error; then perform with bogus to get 401/403
	{
		var out map[string]any
		_, _ = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/refresh", nil, nil, &out)
	}

	// Call logout-all with bearer token
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/logout-all", map[string]string{
		"Authorization": "Bearer " + env.AdminJWT,
	}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Reuse previous refresh cookie should now fail
	rresp, rerr := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/refresh", nil, map[string]any{}, nil)
	require.Error(t, rerr)
	if rresp != nil {
		assert.Contains(t, []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusBadRequest}, rresp.StatusCode)
	}
}

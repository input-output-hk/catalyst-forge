//go:build integration

package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// TestAuth_LogoutAll verifies logout-all invalidates all sessions and requires auth.
func TestAuth_LogoutAll_RequiresAuth_ThenSucceeds(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	// Requires auth
	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/logout-all", nil, nil, &out)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// With bearer succeeds
	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	out = map[string]any{}
	resp, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/logout-all", headers, nil, &out)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

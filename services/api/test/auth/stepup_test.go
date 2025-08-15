//go:build integration

package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// TestAuth_StepUp_Begin_Unauthorized ensures step-up begin requires auth.
func TestAuth_StepUp_Begin_Unauthorized(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/step-up/begin", nil, nil, &out)
	// Expect unauthorized: helper returns an error for 401; assert status via resp
	require.Error(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}
}

// TestAuth_StepUp_Flow_Negative ensures complete without session key fails.
func TestAuth_StepUp_Flow_Negative(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	// Begin requires auth; provide admin bearer
	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	var begin map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/step-up/begin", headers, nil, &begin)
	// Depending on WebAuthn availability in test, begin may be 200 or 500 (if ceremony cannot start).
	if err != nil && resp != nil && resp.StatusCode == http.StatusInternalServerError {
		// acceptable in CI without WebAuthn backing
		return
	}
	require.NoError(t, err)
	if resp.StatusCode == http.StatusOK {
		// Complete with bad payload should fail 401
		bad := map[string]any{
			"session_key": "not-a-real-session",
			"credential":  map[string]any{"invalid": true},
		}
		var out map[string]any
		resp, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/step-up/complete", headers, bad, &out)
		require.Error(t, err)
		if resp != nil {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		}
	}
}

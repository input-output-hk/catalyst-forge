//go:build integration

package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// TestAuth_Login_Begin_Complete_Negatives validates endpoints exist and return proper errors without ceremony.
func TestAuth_Login_Begin_Complete_Negatives(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	// Begin returns 200 with publicKey options
	var begin map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/login/begin", nil, nil, &begin)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Complete without real ceremony should 401
	var complete map[string]any
	resp, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/login/complete", nil, map[string]any{
		"session_key": "bad",
		"credential":  map[string]any{"fake": true},
	}, &complete)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

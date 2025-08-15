//go:build integration

package auth_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuth_JWKS_Smoke ensures the JWKS endpoint is exposed and returns at least one key.
func TestAuth_JWKS_Smoke(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	// GET /.well-known/jwks.json
	req, err := http.NewRequest(http.MethodGet, env.BaseURL()+"/.well-known/jwks.json", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	keys, ok := body["keys"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(keys), 1)
}

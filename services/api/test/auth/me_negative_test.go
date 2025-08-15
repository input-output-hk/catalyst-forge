//go:build integration

package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// TestAuth_Me_Unauthorized ensures /me requires authentication.
func TestAuth_Me_Unauthorized(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	var out map[string]any
	resp, _ := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/me", nil, nil, &out)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

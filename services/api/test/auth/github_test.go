//go:build integration

package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// TestAuth_Github_Exchange_Disabled verifies exchange returns 400 when disabled by config (default in tests).
func TestAuth_Github_Exchange_Disabled(t *testing.T) {
	t.Parallel()

	env := newAuthEnv(t)

	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/oidc/github/exchange", nil, map[string]any{"id_token": "fake"}, &out)
	assert.Error(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestAuth_Github_Policies_CRUD_Minimal exercises basic create/list/get/update/delete.
func TestAuth_Github_Policies_CRUD_Minimal(t *testing.T) {
	// no t.Parallel: uses t.Setenv

	// Enable feature for this test
	t.Setenv("AUTH_GITHUBOIDC_ENABLED", "true")

	env := newAuthEnv(t)

	headers := map[string]string{"Authorization": "Bearer " + env.AdminJWT}
	// Step-up header to satisfy write policy if required by runtime rules
	headers["X-Step-Up"] = "1"

	// Create (requires maintainer permissions). Admin role seeded should include them.
	var created map[string]any
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/auth/oidc/github/policies", headers, map[string]any{
		"repository":   "org/repo",
		"refs":         []string{"refs/heads/main"},
		"environments": []string{"prod"},
		"workflows":    []string{"build"},
		"roles":        []string{"maintainer"},
		"enabled":      true,
	}, &created)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	id, _ := created["id"].(string)
	require.NotEmpty(t, id)

	// List
	var list []map[string]any
	resp, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/oidc/github/policies", headers, nil, &list)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.GreaterOrEqual(t, len(list), 1)

	// Get
	var got map[string]any
	resp, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/oidc/github/policies/"+id, headers, nil, &got)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Update
	resp, err = tu.DoJSON(nil, http.MethodPut, env.BaseURL()+"/api/v1/auth/oidc/github/policies/"+id, headers, map[string]any{
		"refs":         []string{"refs/heads/main"},
		"environments": []string{"prod"},
		"workflows":    []string{"build", "deploy"},
		"roles":        []string{"maintainer"},
		"enabled":      false,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Delete
	resp, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/auth/oidc/github/policies/"+id, headers, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

//go:build integration

package rbac

import (
	"net/http"
	"testing"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Roles negatives ---

func TestRBAC_Roles_Get_NotFound(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)

	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/rbac/roles/does-not-exist", headers, nil, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRBAC_Roles_Create_MissingSlug(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)

	body := map[string]any{"name": "No Slug"}
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/roles", headers, body, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRBAC_Roles_Update_SlugMismatch(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)

	body := map[string]any{"slug": "other", "name": "x"}
	resp, err := tu.DoJSON(nil, http.MethodPut, env.BaseURL()+"/api/v1/rbac/roles/abc", headers, body, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- Bindings negatives ---

func TestRBAC_Bindings_List_MissingParams(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)
	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/rbac/bindings", headers, nil, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRBAC_Bindings_ByScope_InvalidScope(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)
	var out map[string]any
	url := env.BaseURL() + "/api/v1/rbac/bindings/by-scope?scope_type=bad&scope_id=1"
	resp, err := tu.DoJSON(nil, http.MethodGet, url, headers, nil, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRBAC_Bindings_Create_InvalidSubjectType(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)
	body := map[string]any{
		"subject":    map[string]any{"type": "foo", "id": "x"},
		"role_slug":  "r",
		"scope_type": "global",
	}
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/bindings", headers, body, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRBAC_Bindings_Create_InvalidScopeType(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)
	body := map[string]any{
		"subject":    map[string]any{"type": "user", "id": "x"},
		"role_slug":  "r",
		"scope_type": "bad",
	}
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/bindings", headers, body, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRBAC_Bindings_Create_MissingFields(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/bindings", headers, map[string]any{}, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRBAC_Bindings_Delete_InvalidID(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)
	resp, err := tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/rbac/bindings/not-a-uuid", headers, nil, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRBAC_Subject_Bump_InvalidSubjectType(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/subjects/bad/123/bump-version", headers, nil, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- Explain negatives ---

func TestRBAC_Explain_MissingFields(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/explain", headers, map[string]any{}, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRBAC_Explain_InvalidSubjectType(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)
	body := map[string]any{
		"subject":    map[string]any{"type": "foo", "id": "x"},
		"permission": "demo:p",
		"resource":   map[string]any{"type": "any", "id": ""},
	}
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/explain", headers, body, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- Auth gating negative ---

func TestRBAC_Unauthorized_NoAuth(t *testing.T) {
	env := newEnv(t)
	// no auth headers
	var out map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/rbac/roles", nil, nil, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

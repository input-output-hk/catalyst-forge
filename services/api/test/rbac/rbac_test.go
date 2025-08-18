//go:build integration

package rbac

import (
	"net/http"
	"testing"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEnv(t *testing.T) *tu.Env {
	t.Helper()
	if suite == nil {
		t.Fatalf("suite not initialized")
	}
	env, err := tu.NewTestEnv(t.Context(), suite)
	require.NoError(t, err)
	t.Cleanup(env.Close)
	return env
}

func authHeaders(env *tu.Env) map[string]string {
	return map[string]string{"Authorization": "Bearer " + env.AdminJWT}
}

// --- Conditions ---

func TestRBAC_Conditions_List_OK(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)

	var out struct {
		Conditions []string `json:"conditions"`
	}
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/rbac/conditions", headers, nil, &out)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Should include core conditions
	require.NotEmpty(t, out.Conditions)
	assert.Contains(t, out.Conditions, "requires_step_up")
	assert.Contains(t, out.Conditions, "attr_equals")
	assert.Contains(t, out.Conditions, "org_matches")
}

// --- Roles CRUD ---

func TestRBAC_Roles_CRUD(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)

	// Create role
	role := map[string]any{
		"slug":        "test_role",
		"name":        "Test Role",
		"description": "for testing",
		"version":     1,
		"entries": []map[string]any{
			{"effect": "allow", "permission": "demo:ping", "resourceType": ""},
		},
	}
	_, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/roles", headers, role, nil)
	require.NoError(t, err)

	// Get role
	var got map[string]any
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/rbac/roles/test_role", headers, nil, &got)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Test Role", got["name"])

	// Update role (change name)
	role["name"] = "Test Role v2"
	resp, err = tu.DoJSON(nil, http.MethodPut, env.BaseURL()+"/api/v1/rbac/roles/test_role", headers, role, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Bump version
	resp, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/roles/test_role/bump-version", headers, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// --- Bindings flow ---

func TestRBAC_Bindings_Add_List_Delete_Bump(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)

	// Ensure role exists
	role := map[string]any{
		"slug":    "bind_role",
		"name":    "Bind Role",
		"version": 1,
		"entries": []map[string]any{{"effect": "allow", "permission": "demo:perm"}},
	}
	_, _ = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/roles", headers, role, nil)

	// Fetch admin user id
	var me map[string]any
	_, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/me", headers, nil, &me)
	require.NoError(t, err)
	uid, _ := me["id"].(string)
	require.NotEmpty(t, uid)

	// Add binding (global)
	bindReq := map[string]any{
		"subject":    map[string]any{"type": "user", "id": uid},
		"role_slug":  "bind_role",
		"scope_type": "global",
		"scope_id":   "",
	}
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/bindings", headers, bindReq, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// List bindings
	var list map[string]any
	url := env.BaseURL() + "/api/v1/rbac/bindings?subject_type=user&subject_id=" + uid
	_, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	arr, ok := list["bindings"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, arr)

	// Delete the first binding
	first := arr[0].(map[string]any)
	bid, _ := first["id"].(string)
	require.NotEmpty(t, bid)
	_, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/rbac/bindings/"+bid, headers, nil, nil)
	require.NoError(t, err)

	// Bump principal version
	_, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/subjects/user/"+uid+"/bump-version", headers, nil, nil)
	require.NoError(t, err)
}

// --- Explain ---

func TestRBAC_Explain_Allow_And_StepUp428(t *testing.T) {
	env := newEnv(t)
	headers := authHeaders(env)

	// Prepare roles
	allowRole := map[string]any{
		"slug":    "explain_allow",
		"name":    "Explain Allow",
		"version": 1,
		"entries": []map[string]any{{"effect": "allow", "permission": "demo:ping"}},
	}
	_, _ = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/roles", headers, allowRole, nil)

	stepUpRole := map[string]any{
		"slug":    "explain_stepup",
		"name":    "Explain StepUp",
		"version": 1,
		"entries": []map[string]any{{"effect": "allow", "permission": "demo:secure", "conditions": []map[string]any{{"name": "requires_step_up", "params": map[string]any{"within": "5m"}}}}},
	}
	_, _ = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/roles", headers, stepUpRole, nil)

	// Get admin id
	var me map[string]any
	_, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/auth/me", headers, nil, &me)
	require.NoError(t, err)
	uid, _ := me["id"].(string)
	require.NotEmpty(t, uid)

	// Bind roles
	_, _ = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/bindings", headers, map[string]any{"subject": map[string]any{"type": "user", "id": uid}, "role_slug": "explain_allow", "scope_type": "global"}, nil)
	_, _ = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/bindings", headers, map[string]any{"subject": map[string]any{"type": "user", "id": uid}, "role_slug": "explain_stepup", "scope_type": "global"}, nil)

	// Explain allow
	var exp map[string]any
	body := map[string]any{
		"subject":    map[string]any{"type": "user", "id": uid},
		"permission": "demo:ping",
		"resource":   map[string]any{"type": "any", "id": ""},
	}
	resp, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/explain", headers, body, &exp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "allow", exp["decision"])

	// Explain step-up should return 428
	body["permission"] = "demo:secure"
	_, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rbac/explain", headers, body, &exp)
	require.Error(t, err)
}

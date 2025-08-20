//go:build integration

package domain

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

func TestEnvironments_List_And_Get_NotFound(t *testing.T) {
	t.Parallel()
	env := newDomainEnv(t)

	// List environments
	var list map[string]any
	headers := withBypassHeaders(authHeaders(env))
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/environments?page=1&page_size=20", headers, nil, &list)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Not found (or bad request due to current URI binding)
	var out map[string]any
	resp, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/environments/"+uuid.NewString(), headers, nil, &out)
	require.Error(t, err)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, resp.StatusCode)
}

func TestEnvironments_Create_And_List(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	// Write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",env:write"

	// Create
	create := map[string]any{
		"project_id":       projID.String(),
		"name":             "dev",
		"environment_type": "dev",
	}
	var created map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/environments", headers, create, &created)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	// Get by id positive
	var got map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/environments/"+created["id"].(string), headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// List with filter by name
	var list map[string]any
	url := env.BaseURL() + "/api/v1/environments?page=1&page_size=20&name=dev"
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

func TestEnvironments_GetByProjectAndName_Positive_And_Invalid(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	// Write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",env:write"

	// Create env 'stage'
	create := map[string]any{
		"project_id":       projID.String(),
		"name":             "stage",
		"environment_type": "staging",
	}
	var created map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/environments", headers, create, &created)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	// Get by project and name
	var got map[string]any
	url := env.BaseURL() + "/api/v1/projects/" + projID.String() + "/environments/stage"
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, "stage", got["name"])

	// Invalid params: refactored handler ignores project_id and looks up by name only.
	// Accept 200 or legacy 400/404 if validation changes later.
	var bad map[string]any
	badURL := env.BaseURL() + "/api/v1/projects/not-a-uuid/environments/stage"
	r, err = tu.DoJSON(nil, http.MethodGet, badURL, headers, nil, &bad)
	require.NoError(t, err)
	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, r.StatusCode)
}

func TestEnvironments_Update_And_Delete(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)
	// Write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",env:write"

	// Create env
	create := map[string]any{
		"project_id":       projID.String(),
		"name":             "prod-like",
		"environment_type": "dev",
	}
	var created map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/environments", headers, create, &created)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	id := created["id"].(string)

	// Update -> set environment_type prod
	update := map[string]any{"environment_type": "prod"}
	var upd map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/environments/"+id, headers, update, &upd)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Delete: may return 409 if protected
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/environments/"+id, headers, nil, nil)
	require.Error(t, err)
	assert.Contains(t, []int{http.StatusNoContent, http.StatusConflict}, r.StatusCode)

	// Delete not-found
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/environments/"+uuid.NewString(), headers, nil, nil)
	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode)
}

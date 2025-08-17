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

func TestDeployments_List_And_Get_NotFound(t *testing.T) {
	t.Parallel()
	env := newDomainEnv(t)

	headers := withBypassHeaders(authHeaders(env))
	// List empty
	var list map[string]any
	r, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/deployments?page=1&page_size=20", headers, nil, &list)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, r.StatusCode)

	// Get not found or bad request
	var out map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/deployments/"+uuid.NewString(), headers, nil, &out)
	require.Error(t, err)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, r.StatusCode)
}

func TestDeployments_Create_And_List(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	// Need a release and environment to create deployment
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",release:write,env:write"

	// Create environment
	eCreate := map[string]any{
		"project_id":       projID.String(),
		"name":             "dev",
		"environment_type": "dev",
	}
	var envResp map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/environments", headers, eCreate, &envResp)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	envID := envResp["id"].(string)

	// Create release
	rCreate := map[string]any{
		"project_id":    projID.String(),
		"release_key":   "rel-deploy-1",
		"source_commit": "feedface",
	}
	var rel map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases", headers, rCreate, &rel)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	releaseID := rel["id"].(string)

	// Create deployment
	dCreate := map[string]any{
		"release_id":     releaseID,
		"environment_id": envID,
		"deployed_by":    "tester",
	}
	var dep map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/deployments", headers, dCreate, &dep)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	// List by environment and release
	var list map[string]any
	url := env.BaseURL() + "/api/v1/deployments?page=1&page_size=20&environment_id=" + envID + "&release_id=" + releaseID
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

func TestDeployments_GetByID_Positive(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",release:write,env:write,deploy:write"

	// Create environment
	envCreate := map[string]any{
		"project_id":       projID.String(),
		"name":             "dev",
		"environment_type": "dev",
	}
	var e map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/environments", headers, envCreate, &e)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	envID := e["id"].(string)

	// Create release
	relCreate := map[string]any{
		"project_id":    projID.String(),
		"release_key":   "rel-dep",
		"source_commit": "deadbeef",
	}
	var rel map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases", headers, relCreate, &rel)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	relID := rel["id"].(string)

	// Create deployment
	depCreate := map[string]any{
		"release_id":     relID,
		"environment_id": envID,
		"deployed_by":    "tester",
	}
	var dep map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/deployments", headers, depCreate, &dep)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	depID := dep["id"].(string)

	// Get by ID
	var got map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/deployments/"+depID, headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

func TestDeployments_Update_And_Delete(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",release:write,env:write,deploy:write"

	// Create environment
	envCreate := map[string]any{
		"project_id":       projID.String(),
		"name":             "dev-upd",
		"environment_type": "dev",
	}
	var e map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/environments", headers, envCreate, &e)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	envID := e["id"].(string)

	// Create release
	relCreate := map[string]any{
		"project_id":    projID.String(),
		"release_key":   "rel-upd-dep",
		"source_commit": "deadbeef",
	}
	var rel map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases", headers, relCreate, &rel)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	relID := rel["id"].(string)

	// Create deployment
	depCreate := map[string]any{
		"release_id":     relID,
		"environment_id": envID,
	}
	var dep map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/deployments", headers, depCreate, &dep)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	depID := dep["id"].(string)

	// Update deployment: set status and reason
	upd := map[string]any{"status": "rendered", "status_reason": "ok"}
	var out map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/deployments/"+depID, headers, upd, &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Invalid transition -> 422
	bad := map[string]any{"status": "healthy"}
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/deployments/"+depID, headers, bad, &out)
	require.Error(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, r.StatusCode)

	// Delete success
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/deployments/"+depID, headers, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, r.StatusCode)

	// Delete not-found
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/deployments/"+uuid.NewString(), headers, nil, nil)
	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode)
}

//go:build integration

package domain

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

func TestBuilds_List_And_Get_NotFound(t *testing.T) {
	t.Parallel()
	env := newDomainEnv(t)

	// List builds should be 200
	var list map[string]any
	headers := withBypassHeaders(authHeaders(env))
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/builds?page=1&page_size=20", headers, nil, &list)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Not found (or bad request due to current URI binding)
	var out map[string]any
	resp, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/builds/"+uuid.NewString(), headers, nil, &out)
	require.Error(t, err)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, resp.StatusCode)
}

func TestBuilds_Create_And_List(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, projID := seedRepoProject(t)

	// Grant write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",build:write"

	// Create build
	create := map[string]any{
		"repo_id":         repoID.String(),
		"project_id":      projID.String(),
		"commit_sha":      "abc123",
		"branch":          "main",
		"workflow_run_id": "wf-1",
		"status":          "queued",
	}
	var created map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, create, &created)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	id, ok := created["id"].(string)
	require.True(t, ok)
	_, err = uuid.Parse(id)
	require.NoError(t, err)

	// Get by id (positive)
	var got map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/builds/"+id, headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// List with filters to find the created build
	var list map[string]any
	url := env.BaseURL() + "/api/v1/builds?page=1&page_size=20&repo_id=" + repoID.String() + "&project_id=" + projID.String() + "&branch=main"
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	_ = time.Now() // keep import until further expansions
}

func TestBuilds_Create_ErrorCases(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	// Grant write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",build:write"

	// repo not found
	badRepo := map[string]any{
		"repo_id":    uuid.NewString(),
		"project_id": projID.String(),
		"commit_sha": "abc",
		"status":     "queued",
	}
	var out map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, badRepo, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, r.StatusCode)

	// project not found
	repoID2, _ := seedRepoProject(t)
	badProj := map[string]any{
		"repo_id":    repoID2.String(),
		"project_id": uuid.NewString(),
		"commit_sha": "abc",
		"status":     "queued",
	}
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, badProj, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, r.StatusCode)
}

func TestBuilds_Update_And_UpdateStatus(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, projID := seedRepoProject(t)

	// Grant write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",build:write"

	// Create build
	create := map[string]any{
		"repo_id":         repoID.String(),
		"project_id":      projID.String(),
		"commit_sha":      "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		"branch":          "feature/x",
		"workflow_run_id": "wf-update",
		"status":          "queued",
	}
	var created map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, create, &created)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	id := created["id"].(string)

	// Update metadata and status
	update := map[string]any{
		"runner_env": map[string]any{"os": "linux", "arch": "arm64"},
		"status":     "running",
	}
	var upd map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/builds/"+id, headers, update, &upd)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Update status only
	status := map[string]any{"status": "success"}
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/builds/"+id+"/status", headers, status, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, r.StatusCode)
}

func TestBuilds_Update_Invalid_And_NotFound(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, projID := seedRepoProject(t)

	// Grant write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",build:write"

	// Create build
	create := map[string]any{
		"repo_id":    repoID.String(),
		"project_id": projID.String(),
		"commit_sha": "cafebabe",
		"status":     "queued",
	}
	var created map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, create, &created)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	id := created["id"].(string)

	// Invalid status update -> 422
	upd := map[string]any{"status": "success"}
	var out map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/builds/"+id, headers, upd, &out)
	require.Error(t, err)
	require.Equal(t, http.StatusUnprocessableEntity, r.StatusCode)

	// Not found update
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/builds/"+uuid.NewString(), headers, map[string]any{"status": "running"}, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, r.StatusCode)

	// Not found update-status
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/builds/"+uuid.NewString()+"/status", headers, map[string]any{"status": "failed"}, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, r.StatusCode)
}

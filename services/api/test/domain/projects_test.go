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

func TestProjects_List_EmptyAndNotFound(t *testing.T) {
	// serialized
	env := newDomainEnv(t)

	// List projects should return 200 even if empty
	var list map[string]any
	headers := withBypassHeaders(authHeaders(env))
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/projects?page=1&page_size=20", headers, nil, &list)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Not found by random ID (or bad request depending on current binding)
	var out map[string]any
	resp, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/projects/"+uuid.NewString(), headers, nil, &out)
	require.Error(t, err)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, resp.StatusCode)
}

func TestProjects_Positive_GetByID_ByRepoPath_And_Filters(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, projID := seedRepoProject(t) // seeds repo github.com/acme/demo and project app
	headers := withBypassHeaders(authHeaders(env))

	// Get by id
	var got map[string]any
	r, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/projects/"+projID.String(), headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, repoID.String(), got["repo_id"])
	assert.Equal(t, "app", got["slug"])
	assert.Equal(t, "app", got["path"])

	// Get by repo and path
	var byPath map[string]any
	url := env.BaseURL() + "/api/v1/repositories/" + repoID.String() + "/projects/by-path?path=app"
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &byPath)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, "app", byPath["slug"])

	// List with filters
	var list map[string]any
	furl := env.BaseURL() + "/api/v1/projects?page=1&page_size=20&repo_id=" + repoID.String() + "&path=app&slug=app&status=active"
	r, err = tu.DoJSON(nil, http.MethodGet, furl, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

func TestProjects_ByRepoPath_Invalid_And_NotFound_And_ListFilters(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, _ := seedRepoProject(t)
	headers := withBypassHeaders(authHeaders(env))

	// invalid params: bad repo uuid
	var out map[string]any
	r, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/repositories/not-a-uuid/projects/by-path?path=app", headers, nil, &out)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, r.StatusCode)

	// not found
	url := env.BaseURL() + "/api/v1/repositories/" + repoID.String() + "/projects/by-path?path=does-not-exist"
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &out)
	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode)

	// list filters coverage
	var list map[string]any
	lf := env.BaseURL() + "/api/v1/projects?page=1&page_size=20&repo_id=" + repoID.String() + "&status=active"
	r, err = tu.DoJSON(nil, http.MethodGet, lf, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

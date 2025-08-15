//go:build integration

package domain

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

func TestRepositories_GetByID_And_List(t *testing.T) {
	// serialized

	env := newDomainEnv(t)

	// List repositories (likely empty) should return 200
	headers := withBypassHeaders(authHeaders(env))
	var list map[string]any
	// Provide pagination to satisfy validation
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/repositories?page=1&page_size=20", headers, nil, &list)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRepositories_GetByPath_NotFound_And_InvalidID(t *testing.T) {
	// serialized

	env := newDomainEnv(t)

	// Not found by path
	var out map[string]any
	headers := withBypassHeaders(authHeaders(env))
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/repositories/by-path/github.com/nope/nada", headers, nil, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRepositories_Positive_GetByID_And_ByPath_And_Filters(t *testing.T) {
	// no t.Parallel; uses DB seeding directly
	env := newDomainEnv(t)
	repoID, _ := seedRepoProject(t) // seeds github.com/acme/demo
	headers := withBypassHeaders(authHeaders(env))

	// Get by id
	var got map[string]any
	r, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/repositories/"+repoID.String(), headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, "github.com", got["host"])
	assert.Equal(t, "acme", got["org"])
	assert.Equal(t, "demo", got["name"])

	// Get by path
	var byPath map[string]any
	path := env.BaseURL() + "/api/v1/repositories/by-path/github.com/acme/demo"
	r, err = tu.DoJSON(nil, http.MethodGet, path, headers, nil, &byPath)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// List with filters
	var list map[string]any
	url := env.BaseURL() + "/api/v1/repositories?page=1&page_size=20&host=github.com&org=acme&name=demo"
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

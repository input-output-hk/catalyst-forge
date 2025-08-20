//go:build integration

package domain

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

func TestRenderedReleases_CRUD_And_ByDeployment(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	// Permissions for creating env, release, deployment
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",env:write,release:write,deploy:write"

	// Create environment
	eCreate := map[string]any{
		"project_id":       projID.String(),
		"name":             "rr-dev",
		"environment_type": "dev",
	}
	var e map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/environments", headers, eCreate, &e)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	envID := e["id"].(string)

	// Create release
	relCreate := map[string]any{
		"project_id":    projID.String(),
		"release_key":   "rr-rel-1",
		"source_commit": "cafebabe",
	}
	var rel map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases", headers, relCreate, &rel)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	relID := rel["id"].(string)

	// Create deployment
	dCreate := map[string]any{
		"release_id":     relID,
		"environment_id": envID,
		"deployed_by":    "tester",
	}
	var dep map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/deployments", headers, dCreate, &dep)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	depID := dep["id"].(string)

	// Create rendered release
	rrCreate := map[string]any{
		"deployment_id":    depID,
		"release_id":       relID,
		"environment_id":   envID,
		"renderer_version": "1.0.0",
		"bundle_hash":      "bundlehash",
		"output_hash":      "outputhash",
		"oci_ref":          "ghcr.io/acme/demo:1",
		"oci_digest":       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	var rr map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/rendered-releases", headers, rrCreate, &rr)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	rrID := rr["id"].(string)

	// Get by ID
	var got map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/rendered-releases/"+rrID, headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// List by deployment filter
	var list map[string]any
	url := env.BaseURL() + "/api/v1/rendered-releases?page=1&page_size=20&deployment_id=" + depID
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Get by deployment convenience route
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/deployments/"+depID+"/rendered-release", headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Update rendered release
	upd := map[string]any{"storage_uri": "s3://bucket/path"}
	var out map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/rendered-releases/"+rrID, headers, upd, &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Delete
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/rendered-releases/"+rrID, headers, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, r.StatusCode)
}

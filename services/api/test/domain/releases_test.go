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

func TestReleases_List_And_Get_NotFound(t *testing.T) {
	t.Parallel()
	env := newDomainEnv(t)

	// List releases
	var list map[string]any
	headers := withBypassHeaders(authHeaders(env))
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/releases?page=1&page_size=20", headers, nil, &list)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Not found (or bad request depending on binding)
	var out map[string]any
	resp, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/releases/"+uuid.NewString(), headers, nil, &out)
	require.Error(t, err)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, resp.StatusCode)
}

func TestReleases_Create_And_List(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",release:write"

	// Create release (minimal)
	create := map[string]any{
		"project_id":    projID.String(),
		"release_key":   "rel-1",
		"source_commit": "cafebabe",
	}
	var rel map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases", headers, create, &rel)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	// List by project_id
	var list map[string]any
	url := env.BaseURL() + "/api/v1/releases?page=1&page_size=20&project_id=" + projID.String() + "&release_key=rel-1"
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

func TestReleases_Update_MinimalField(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",release:write"

	// Create release
	create := map[string]any{
		"project_id":    projID.String(),
		"release_key":   "rel-upd",
		"source_commit": "deadbeef",
	}
	var rel map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases", headers, create, &rel)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	id := rel["id"].(string)

	// Update oci_ref
	update := map[string]any{"oci_ref": "ghcr.io/acme/demo:rel-upd"}
	var upd map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/releases/"+id, headers, update, &upd)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

func TestReleases_Subresources_Modules_Injections_Artifacts_And_Delete(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, projID := seedRepoProject(t)

	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",release:write,artifact:write,build:write"

	// Create release
	relCreate := map[string]any{
		"project_id":    projID.String(),
		"release_key":   "rel-sub",
		"source_commit": "cafebabe",
	}
	var rel map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases", headers, relCreate, &rel)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	relID := rel["id"].(string)

	// Modules: add
	mods := map[string]any{
		"modules": []map[string]any{{
			"module_key":  "core",
			"name":        "core",
			"module_type": "helm",
			"version":     "1.0.0",
		}},
	}
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases/"+relID+"/modules", headers, mods, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	// Modules: list
	var modList []map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/releases/"+relID+"/modules", headers, nil, &modList)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	// Modules: remove
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/releases/"+relID+"/modules/core", headers, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, r.StatusCode)

	// Injections: add
	injCreate := map[string]any{
		"injections": []map[string]any{{
			"json_pointer":   "/spec/template",
			"artifact_key":   "image",
			"artifact_field": "image_digest",
			"module_key":     "core",
			"module_name":    "core",
		}},
	}
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases/"+relID+"/injections", headers, injCreate, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	// Injections: list then remove first
	var injList []map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/releases/"+relID+"/injections", headers, nil, &injList)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	if len(injList) > 0 {
		injID := injList[0]["id"].(string)
		r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/releases/"+relID+"/injections/"+injID, headers, nil, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, r.StatusCode)
	}

	// Artifacts: create build and artifact, attach, list, detach
	bcreate := map[string]any{
		"repo_id":    repoID.String(),
		"project_id": projID.String(),
		"commit_sha": "feedfacefeedfacefeedfacefeedfacefeedface",
		"status":     "queued",
	}
	var b map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, bcreate, &b)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	aCreate := map[string]any{
		"build_id":     b["id"].(string),
		"project_id":   projID.String(),
		"image_name":   "ghcr.io/acme/demo",
		"image_digest": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"tag":          "v1",
	}
	var a map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/artifacts", headers, aCreate, &a)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	artifactID := a["id"].(string)

	attach := map[string]any{
		"artifact_id":  artifactID,
		"artifact_key": "image",
		"role":         "primary",
	}
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases/"+relID+"/artifacts", headers, attach, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	// List
	var ra []map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/releases/"+relID+"/artifacts", headers, nil, &ra)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	// Detach: include role in path per route
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/releases/"+relID+"/artifacts/"+artifactID+"/primary", headers, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, r.StatusCode)

	// Delete release positive
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/releases/"+relID, headers, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, r.StatusCode)
}

func TestReleases_Delete_Sealed_Conflict_And_Filters(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",release:write"

	// Create release
	relCreate := map[string]any{
		"project_id":    projID.String(),
		"release_key":   "rel-seal",
		"source_commit": "beadfeed",
	}
	var rel map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases", headers, relCreate, &rel)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	relID := rel["id"].(string)

	// Update to sealed
	upd := map[string]any{"status": "sealed"}
	var out map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/releases/"+relID, headers, upd, &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Try to delete -> 409 sealed
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/releases/"+relID, headers, nil, nil)
	require.Error(t, err)
	require.Equal(t, http.StatusConflict, r.StatusCode)

	// List filters
	var list map[string]any
	url := env.BaseURL() + "/api/v1/releases?page=1&page_size=20&project_id=" + projID.String() + "&release_key=rel-seal&status=sealed"
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

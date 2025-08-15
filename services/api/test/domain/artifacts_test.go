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

func TestArtifacts_List_And_Get_NotFound(t *testing.T) {
	t.Parallel()
	env := newDomainEnv(t)

	// List artifacts should be 200
	var list map[string]any
	// Domain endpoints require auth; provide admin bearer and bypass headers (RBAC)
	headers := withBypassHeaders(authHeaders(env))
	resp, err := tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/artifacts?page=1&page_size=20", headers, nil, &list)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Not found (or bad request due to current URI binding)
	var out map[string]any
	resp, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/artifacts/"+uuid.NewString(), headers, nil, &out)
	require.Error(t, err)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, resp.StatusCode)
}

func TestArtifacts_Create_And_List(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, projID := seedRepoProject(t)

	// Grant write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",artifact:write,build:write"

	// Create a build to attach artifact to
	bcreate := map[string]any{
		"repo_id":         repoID.String(),
		"project_id":      projID.String(),
		"commit_sha":      "deadbeef",
		"branch":          "main",
		"workflow_run_id": "wf-a",
		"status":          "queued",
	}
	var b map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, bcreate, &b)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	buildID := b["id"].(string)

	// Create artifact
	digest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcd"
	aCreate := map[string]any{
		"build_id":     buildID,
		"project_id":   projID.String(),
		"image_name":   "ghcr.io/acme/demo",
		"image_digest": digest,
		"tag":          "v1",
		"provider":     "ghcr",
	}
	var a map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/artifacts", headers, aCreate, &a)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	// List filter by build_id
	var list map[string]any
	url := env.BaseURL() + "/api/v1/artifacts?page=1&page_size=20&build_id=" + buildID + "&image_name=ghcr.io/acme/demo&image_digest=" + digest
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

func TestArtifacts_Create_ErrorCases(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	// Grant write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",artifact:write"

	// build not found
	aBad := map[string]any{
		"build_id":     uuid.NewString(),
		"project_id":   uuid.NewString(),
		"image_name":   "img",
		"image_digest": "sha256:bad",
	}
	var out map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/artifacts", headers, aBad, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, r.StatusCode)

	// invalid body
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/artifacts", headers, map[string]any{"build_id": "not-a-uuid"}, &out)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, r.StatusCode)
}

func TestArtifacts_GetByDigest_Positive(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, projID := seedRepoProject(t)

	// Grant write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",artifact:write,build:write"

	// Create a build
	bcreate := map[string]any{
		"repo_id":         repoID.String(),
		"project_id":      projID.String(),
		"commit_sha":      "cafebabe",
		"branch":          "main",
		"workflow_run_id": "wf-digest",
		"status":          "queued",
	}
	var b map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, bcreate, &b)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	// Create an artifact with known digest
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	aCreate := map[string]any{
		"build_id":     b["id"].(string),
		"project_id":   projID.String(),
		"image_name":   "ghcr.io/acme/demo",
		"image_digest": digest,
	}
	var a map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/artifacts", headers, aCreate, &a)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	// Get by digest should be 200
	var got map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/artifacts/digest/"+digest, headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

func TestArtifacts_GetByID_Update_And_Delete(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, projID := seedRepoProject(t)

	// Grant write permissions
	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",artifact:write,build:write"

	// Create a build
	bcreate := map[string]any{
		"repo_id":         repoID.String(),
		"project_id":      projID.String(),
		"commit_sha":      "facefeedfacefeedfacefeedfacefeedfacefeedfacefeed",
		"branch":          "main",
		"workflow_run_id": "wf-art",
		"status":          "queued",
	}
	var b map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, bcreate, &b)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	buildID := b["id"].(string)

	// Create an artifact
	digest := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	acreate := map[string]any{
		"build_id":     buildID,
		"project_id":   projID.String(),
		"image_name":   "ghcr.io/acme/demo",
		"image_digest": digest,
		"tag":          "v2",
	}
	var a map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/artifacts", headers, acreate, &a)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	artifactID := a["id"].(string)

	// Get by ID
	var got map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/artifacts/"+artifactID, headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Update tag and scan status
	update := map[string]any{
		"tag":         "v2.1",
		"scan_status": "passed",
	}
	var upd map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/artifacts/"+artifactID, headers, update, &upd)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Delete positive
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/artifacts/"+artifactID, headers, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, r.StatusCode)

	// Delete not-found
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/artifacts/"+uuid.NewString(), headers, nil, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusNotFound, r.StatusCode)
}

func TestArtifacts_Delete_Conflict_When_Attached_To_Release(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, projID := seedRepoProject(t)

	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",artifact:write,build:write,release:write"

	// Create build
	bcreate := map[string]any{
		"repo_id":    repoID.String(),
		"project_id": projID.String(),
		"commit_sha": "abcdabcdabcdabcdabcdabcdabcdabcdabcdabcd",
		"status":     "queued",
	}
	var b map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, bcreate, &b)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	// Create artifact
	digest := "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	acreate := map[string]any{
		"build_id":     b["id"].(string),
		"project_id":   projID.String(),
		"image_name":   "ghcr.io/acme/demo",
		"image_digest": digest,
		"tag":          "v3",
	}
	var a map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/artifacts", headers, acreate, &a)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	artifactID := a["id"].(string)

	// Create release and attach artifact
	relCreate := map[string]any{
		"project_id":    projID.String(),
		"release_key":   "rel-art",
		"source_commit": "cafebabe",
	}
	var rel map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases", headers, relCreate, &rel)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	relID := rel["id"].(string)

	attach := map[string]any{
		"artifact_id":  artifactID,
		"artifact_key": "image",
		"role":         "primary",
	}
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases/"+relID+"/artifacts", headers, attach, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	// Attempt to delete artifact -> expect 409
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/artifacts/"+artifactID, headers, nil, nil)
	require.Error(t, err)
	require.Equal(t, http.StatusConflict, r.StatusCode)
}

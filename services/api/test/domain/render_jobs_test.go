//go:build integration

package domain

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

func TestRenderJob_FullLifecycle(t *testing.T) {
	// no t.Parallel; uses DB seed directly and shared created resources
	env := newDomainEnv(t)
	_, projID := seedRepoProject(t)

	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",release:write,env:write,deploy:write"

	// Create environment
	eCreate := map[string]any{
		"project_id":       projID.String(),
		"name":             "dev-rj",
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
		"release_key":   "rel-render-1",
		"source_commit": "cafebabe",
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

	// GET render-job before creation -> 404
	var out map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/deployments/"+depID+"/render-job", headers, nil, &out)
	require.Error(t, err)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, r.StatusCode)

	// POST render-job (create)
	rjCreate := map[string]any{
		"module_versions":  []map[string]any{{"name": "workload", "version": "1.0.0"}},
		"bundle_hash":      "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"renderer_version": "1.2.3",
	}
	var rj map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/deployments/"+depID+"/render-job", headers, rjCreate, &rj)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	rjID := rj["id"].(string)
	_, err = uuid.Parse(rjID)
	require.NoError(t, err)

	// GET render-job -> 200
	var got map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/deployments/"+depID+"/render-job", headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, depID, got["deployment_id"])

	// PATCH render-job (running + metadata update)
	now := time.Now().UTC()
	rjUpdate := map[string]any{
		"status":                "running",
		"module_versions":       []map[string]any{{"name": "workload", "version": "1.1.0"}},
		"bundle_hash":           "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		"output_hash":           "sha256:3333333333333333333333333333333333333333333333333333333333333333",
		"storage_uri":           "s3://bucket/path",
		"renderer_version":      "1.2.4",
		"oci_ref":               "ghcr.io/acme/demo@sha256:abc",
		"oci_digest":            "sha256:abc",
		"signed":                true,
		"signature_verified_at": now,
	}
	var upd map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/deployments/"+depID+"/render-job", headers, rjUpdate, &upd)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, "running", upd["status"])

	// PATCH render-job (succeeded + finished_at)
	done := time.Now().UTC()
	rjUpdate2 := map[string]any{
		"status":      "succeeded",
		"finished_at": done,
	}
	var upd2 map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/deployments/"+depID+"/render-job", headers, rjUpdate2, &upd2)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, "succeeded", upd2["status"])

	// Verify deployment status transitioned to pushed
	var depGot map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/deployments/"+depID, headers, nil, &depGot)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, "pushed", depGot["status"])

	// POST render-job again for same deployment -> 409 conflict
	var out2 map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/deployments/"+depID+"/render-job", headers, rjCreate, &out2)
	require.Error(t, err)
	require.Equal(t, http.StatusConflict, r.StatusCode)

	// Negative: POST with non-existent deployment -> 404
	fakeDepID := uuid.NewString()
	rjBad := map[string]any{
		"renderer_version": "1.0.0",
	}
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/deployments/"+fakeDepID+"/render-job", headers, rjBad, &out)
	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode)

	// Negative: PATCH when no render-job exists for another deployment -> 404
	// Create another deployment without render job
	depCreate2 := map[string]any{
		"release_id":     relID,
		"environment_id": envID,
	}
	var dep2 map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/deployments", headers, depCreate2, &dep2)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	depID2 := dep2["id"].(string)

	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/deployments/"+depID2+"/render-job", headers, map[string]any{"status": "running"}, &out)
	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode)

	// Negative: PATCH invalid status value -> 400
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/deployments/"+depID+"/render-job", headers, map[string]any{"status": "wrong"}, &out)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, r.StatusCode)

	// Negative: POST invalid body (malformed JSON) -> 400
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPost, env.BaseURL()+"/api/v1/deployments/"+depID+"/render-job", bytes.NewBufferString("{invalid"))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

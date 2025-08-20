//go:build integration

package domain

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

func TestPromotions_CRUD(t *testing.T) {
	// no t.Parallel; uses DB seed directly
	env := newDomainEnv(t)
	repoID, projID := seedRepoProject(t)

	headers := withBypassHeaders(authHeaders(env))
	headers["X-Test-Permissions"] = headers["X-Test-Permissions"] + ",env:write,release:write"

	// Create environment
	eCreate := map[string]any{
		"project_id":       projID.String(),
		"name":             "promo-dev",
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
		"release_key":   "promo-rel",
		"source_commit": "feedface",
	}
	var rel map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/releases", headers, relCreate, &rel)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	relID := rel["id"].(string)

	// Create a build and artifact to ensure project has activity (optional, aligns with other tests)
	bCreate := map[string]any{
		"repo_id":    repoID.String(),
		"project_id": projID.String(),
		"commit_sha": "feedfacefeedfacefeedfacefeedfacefeedface",
		"status":     "queued",
	}
	var b map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/builds", headers, bCreate, &b)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)

	// Create promotion (manual)
	pCreate := map[string]any{
		"project_id":     projID.String(),
		"release_id":     relID,
		"environment_id": envID,
		"approval_mode":  "manual",
		"requested_by":   "approver",
		"reason":         "promote to dev",
		"policy_results": map[string]any{"lint": "ok"},
	}
	var p map[string]any
	r, err = tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/promotions", headers, pCreate, &p)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	pID := p["id"].(string)

	// Get by ID
	var got map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/promotions/"+pID, headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// List by filters
	var list map[string]any
	url := env.BaseURL() + "/api/v1/promotions?page=1&page_size=20&project_id=" + projID.String() + "&environment_id=" + envID + "&release_id=" + relID
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Update status to approved with approver
	upd := map[string]any{"status": "approved", "approver_id": "admin", "reason": "looks good"}
	var out map[string]any
	r, err = tu.DoJSON(nil, http.MethodPatch, env.BaseURL()+"/api/v1/promotions/"+pID, headers, upd, &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)

	// Delete
	r, err = tu.DoJSON(nil, http.MethodDelete, env.BaseURL()+"/api/v1/promotions/"+pID, headers, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, r.StatusCode)
}

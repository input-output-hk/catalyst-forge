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

func TestTraces_Create_Get_List(t *testing.T) {
	env := newDomainEnv(t)
	headers := withBypassHeaders(authHeaders(env))

	// Create trace (with repo_id omitted)
	create := map[string]any{
		"purpose":         "build",
		"retention_class": "short",
		"branch":          "main",
		"created_by":      "tester",
	}
	var tr map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/traces", headers, create, &tr)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, r.StatusCode)
	id := tr["id"].(string)
	_, err = uuid.Parse(id)
	require.NoError(t, err)

	// Get by ID
	var got map[string]any
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/traces/"+id, headers, nil, &got)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	assert.Equal(t, "build", got["purpose"])
	assert.Equal(t, "short", got["retention_class"])

	// List with filters
	since := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	until := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	var list map[string]any
	url := env.BaseURL() + "/api/v1/traces?page=1&page_size=20&purpose=build&retention_class=short&branch=main&created_by=tester&since=" + since + "&until=" + until
	r, err = tu.DoJSON(nil, http.MethodGet, url, headers, nil, &list)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
}

func TestTraces_Invalid_And_NotFound(t *testing.T) {
	env := newDomainEnv(t)
	headers := withBypassHeaders(authHeaders(env))

	// Invalid create (bad retention_class)
	bad := map[string]any{
		"purpose":         "build",
		"retention_class": "ephemeral",
	}
	var out map[string]any
	r, err := tu.DoJSON(nil, http.MethodPost, env.BaseURL()+"/api/v1/traces", headers, bad, &out)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, r.StatusCode)

	// Get invalid id
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/traces/not-a-uuid", headers, nil, &out)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, r.StatusCode)

	// Get not found
	r, err = tu.DoJSON(nil, http.MethodGet, env.BaseURL()+"/api/v1/traces/"+uuid.NewString(), headers, nil, &out)
	require.Error(t, err)
	assert.Contains(t, []int{http.StatusNotFound, http.StatusBadRequest}, r.StatusCode)
}

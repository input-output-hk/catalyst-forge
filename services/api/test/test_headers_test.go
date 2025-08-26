//go:build integration

package test

import (
	"net/http"
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/require"
)

// TestHeadersEndpoint verifies /api/v1/test returns 200 and echoes headers
func TestHeadersEndpoint(t *testing.T) {
	env := NewTestEnv(t)
	baseURL := env.BaseURL()

	client := &http.Client{}
	req, err := http.NewRequest("GET", baseURL+"/api/v1/test", nil)
	require.NoError(t, err)
	req.Header.Set("X-Demo", "demo-value")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Headers map[string][]string `json:"headers"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Contains(t, body.Headers, "X-Demo")
	require.Contains(t, body.Headers["X-Demo"], "demo-value")
}

//go:build integration

package test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthEndpoint is kept as a simple smoke test.
// Note: The test harness already waits for /healthz during server startup (see testutil/server.go:waitForHealthy),
// so this test primarily serves as a basic connectivity check and could be considered redundant.
// Decision: Keep as a minimal smoke test to ensure the endpoint remains accessible after initialization.
func TestHealthEndpoint(t *testing.T) {
    env := NewTestEnv(t)
	apiURL := env.BaseURL()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create an HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/healthz", apiURL), nil)
	require.NoError(t, err)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Check the response status
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected successful health check")
}

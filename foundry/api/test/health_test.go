package test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create an HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/healthz", apiURL), nil)
	require.NoError(t, err)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check the response status
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected successful health check")
}

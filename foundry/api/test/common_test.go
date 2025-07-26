package test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

// newTestClient creates a new client for testing with default configuration
func newTestClient() client.Client {
	// Get JWT token path from environment variable
	tokenPath := os.Getenv("JWT_TOKEN_PATH")
	if tokenPath == "" {
		panic("JWT_TOKEN_PATH environment variable is required")
	}

	// Read token from file
	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		panic(fmt.Sprintf("failed to read JWT token from %s: %v", tokenPath, err))
	}

	// Trim whitespace and check if token is present
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		panic(fmt.Sprintf("JWT token file %s is empty", tokenPath))
	}

	return client.NewClient(getTestAPIURL(), client.WithToken(token))
}

// newTestContext creates a new context with timeout for testing
func newTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// getTestAPIURL returns the API URL for testing
func getTestAPIURL() string {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}
	return apiURL
}

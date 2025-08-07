package test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/releases"
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

// generateTestName creates a unique test name with timestamp
func generateTestName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().Unix())
}

// generateTestEmail generates a unique test email
func generateTestEmail() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("test-user-%x@example.com", bytes)
}

// generateTestKid generates a unique test key ID
func generateTestKid() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("test-key-%x", bytes)
}

// generateTestPubKey generates a test public key
func generateTestPubKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

// createTestRelease creates a test release with common defaults
func createTestRelease(client client.Client, ctx context.Context, projectName string) (*releases.Release, error) {
	bundleStr := base64.StdEncoding.EncodeToString([]byte("test bundle data"))
	release := &releases.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "abcdef123456",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}
	return client.Releases().Create(ctx, release, false)
}

// stringPtr returns a pointer to a string (helper for optional fields)
func stringPtr(s string) *string {
	return &s
}

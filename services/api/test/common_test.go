//go:build integration

package test

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "time"

    "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/releases"
)

// newTestContext creates a new context with timeout for testing
func newTestContext() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), 30*time.Second)
}

// generateTestName creates a unique test name with timestamp
func generateTestName(prefix string) string {
    return fmt.Sprintf("%s-%d", prefix, time.Now().Unix())
}

// generateTestEmail generates a unique test email
func generateTestEmail() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// fallback to time-based bytes to satisfy linter; acceptable for tests
		for i := range bytes {
			bytes[i] = byte(time.Now().UnixNano() >> (i % 8))
		}
	}
	return fmt.Sprintf("test-user-%x@example.com", bytes)
}

// generateTestKid generates a unique test key ID
func generateTestKid() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		for i := range bytes {
			bytes[i] = byte(time.Now().UnixNano() >> (i % 8))
		}
	}
	return fmt.Sprintf("test-key-%x", bytes)
}

// generateTestPubKey generates a test public key
func generateTestPubKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		for i := range bytes {
			bytes[i] = byte(time.Now().UnixNano() >> (i % 8))
		}
	}
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

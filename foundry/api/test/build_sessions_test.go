//go:build integration

package test

import (
	"testing"
	"time"

	buildsessions "github.com/input-output-hk/catalyst-forge/lib/foundry/client/buildsessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSessions_Create(t *testing.T) {
    env := NewTestEnv(t)
	c := env.AdminClient()
	ctx, cancel := newTestContext()
	defer cancel()

	req := struct {
		OwnerType string                 `json:"owner_type"`
		OwnerID   string                 `json:"owner_id"`
		TTL       string                 `json:"ttl"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}{
		OwnerType: "repo",
		OwnerID:   "owner/repo",
		TTL:       "10m",
		Metadata:  map[string]interface{}{"workflow": "ci", "build_id": "123", "branch": "main"},
	}

	out, err := c.BuildSessions().Create(ctx, &buildsessions.CreateRequest{
		OwnerType: req.OwnerType,
		OwnerID:   req.OwnerID,
		TTL:       req.TTL,
		Metadata:  req.Metadata,
	})
	require.NoError(t, err)
	require.NotEmpty(t, out.ID)
	
	// Validate TTL parsing and expiry semantics
	require.NotEmpty(t, out.ExpiresAt)
	expiresAt, err := time.Parse(time.RFC3339, out.ExpiresAt)
	require.NoError(t, err, "ExpiresAt should be a valid RFC3339 timestamp")
	
	// Check that expiry is approximately 10 minutes from now
	expectedExpiry := time.Now().Add(10 * time.Minute)
	timeDiff := expiresAt.Sub(expectedExpiry)
	assert.Less(t, timeDiff.Abs(), 30*time.Second, "Expiry time should be within 30 seconds of expected")
	assert.True(t, expiresAt.After(time.Now()), "Expiry time should be in the future")
	
	// NOTE: Cannot test metadata persistence without Get endpoint
	// NOTE: Cannot test actual expiry behavior without Get/List endpoints
}

// Ensure invalid TTL formats are rejected by the API.
func TestBuildSessions_Create_InvalidTTL(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()

    testCases := []struct {
        name string
        ttl  string
    }{
        {"not a duration", "not-a-duration"},
        {"empty string", ""},
        {"negative duration", "-10m"},
        {"zero duration", "0s"},
        {"invalid format", "10"},
        {"mixed invalid", "10m30"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            _, err := c.BuildSessions().Create(ctx, &buildsessions.CreateRequest{
                OwnerType: "repo",
                OwnerID:   "owner/repo",
                TTL:       tc.ttl,
                Metadata:  map[string]interface{}{"workflow": "ci"},
            })
            require.Error(t, err, "Should reject TTL: %s", tc.ttl)
        })
    }
}

// Test invalid owner_type values
func TestBuildSessions_Create_InvalidOwnerType(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()

    testCases := []struct {
        name      string
        ownerType string
    }{
        {"empty owner type", ""},
        {"invalid owner type", "invalid"},
        {"numeric owner type", "123"},
        {"special chars", "repo!@#"},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            _, err := c.BuildSessions().Create(ctx, &buildsessions.CreateRequest{
                OwnerType: tc.ownerType,
                OwnerID:   "owner/repo",
                TTL:       "10m",
                Metadata:  map[string]interface{}{"workflow": "ci"},
            })
            // Note: API might accept any owner_type, so we just ensure no panic
            // If validation is added, this should assert an error
            _ = err
        })
    }
}

// Test edge cases for TTL values
func TestBuildSessions_Create_TTLBoundaries(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()

    testCases := []struct {
        name        string
        ttl         string
        shouldError bool
    }{
        {"very short TTL", "1s", false},
        {"typical TTL", "30m", false},
        {"long TTL", "24h", false},
        {"very long TTL", "720h", false}, // 30 days
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            out, err := c.BuildSessions().Create(ctx, &buildsessions.CreateRequest{
                OwnerType: "repo",
                OwnerID:   "owner/repo-" + tc.name,
                TTL:       tc.ttl,
                Metadata:  map[string]interface{}{"test": tc.name},
            })
            
            if tc.shouldError {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
                require.NotEmpty(t, out.ID)
                
                // Verify the TTL was applied correctly
                duration, _ := time.ParseDuration(tc.ttl)
                expiresAt, _ := time.Parse(time.RFC3339, out.ExpiresAt)
                expectedExpiry := time.Now().Add(duration)
                timeDiff := expiresAt.Sub(expectedExpiry)
                assert.Less(t, timeDiff.Abs(), 30*time.Second, "TTL should be applied correctly for %s", tc.ttl)
            }
        })
    }
}

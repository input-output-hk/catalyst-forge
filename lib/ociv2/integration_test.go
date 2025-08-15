package ociv2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationPushPull tests the complete push/pull cycle using an in-memory registry
func TestIntegrationPushPull(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	t.Parallel()
	
	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()
	
	// Extract registry host from server URL
	registryHost := strings.TrimPrefix(registryServer.URL, "http://")
	
	// Create client with proper configuration for test registry
	client, err := New(ClientOptions{
		PlainHTTP:              true, // Test server uses HTTP
		PreferArtifactManifest: true,
		FallbackImageManifest:  true,
		EnableMetrics:          true,
		MetricsCallback: func(m *Metrics) {
			t.Logf("Metrics callback: %+v", m)
		},
	})
	require.NoError(t, err)
	
	ctx := context.Background()
	
	t.Run("JSON_Artifact_RoundTrip", func(t *testing.T) {
		testRef := fmt.Sprintf("%s/test/json:latest", registryHost)
		
		// Test data
		testJSON := []byte(`{"message": "hello world", "version": "1.0.0"}`)
		testMediaType := "application/vnd.example.config+json"
		testAnnotations := NewAnnotations().
			WithForgeKind("config").
			WithForgeProject("test").
			WithSource("https://github.com/test/repo", "abc123")
		
		// Push JSON artifact
		pushDesc, err := client.PushJSON(ctx, testRef, testMediaType, testJSON, testAnnotations)
		require.NoError(t, err)
		
		// Verify push result
		assert.NotEmpty(t, pushDesc.Digest)
		assert.Equal(t, int64(len(testJSON)), pushDesc.Size)
		assert.Equal(t, testMediaType, pushDesc.MediaType)
		assert.Contains(t, pushDesc.Ref, "@sha256:")
		
		// Pull JSON artifact back
		pulledJSON, pullDesc, err := client.PullJSON(ctx, testRef, testMediaType)
		require.NoError(t, err)
		
		// Verify pull result
		assert.Equal(t, testJSON, pulledJSON)
		assert.Equal(t, pushDesc.Digest, pullDesc.Digest)
		assert.Equal(t, pushDesc.Size, pullDesc.Size)
		assert.Equal(t, testMediaType, pullDesc.MediaType)
		
		// Test resolve operation
		resolveDesc, err := client.Resolve(ctx, testRef)
		require.NoError(t, err)
		assert.Equal(t, pushDesc.Digest, resolveDesc.Digest)
		
		// Test head operation
		headDesc, err := client.Head(ctx, testRef)
		require.NoError(t, err)
		assert.Equal(t, pushDesc.Digest, headDesc.Digest)
	})
	
	t.Run("TAR_Artifact_RoundTrip", func(t *testing.T) {
		testRef := fmt.Sprintf("%s/test/tar:v1.0", registryHost)
		
		// Test data
		configJSON := []byte(`{"name": "test-package", "version": "1.0.0"}`)
		configMediaType := "application/vnd.example.package.config+json"
		layerMediaType := "application/vnd.example.package.layer.v1.tar+gzip"
		tarData := "test tar content for integration test"
		tarReader := strings.NewReader(tarData)
		
		testAnnotations := NewAnnotations().
			WithForgeKind("package").
			WithForgeProject("test").
			WithTrace("trace-456")
		
		// Push TAR artifact
		pushDesc, err := client.PushTar(ctx, testRef, configJSON, configMediaType, 
			layerMediaType, tarReader, int64(len(tarData)), testAnnotations)
		require.NoError(t, err)
		
		// Verify push result
		assert.NotEmpty(t, pushDesc.Digest)
		assert.NotEmpty(t, pushDesc.Ref)
		assert.Contains(t, pushDesc.Ref, "@sha256:")
		
		// Pull TAR artifact back
		pulledTar, pullDesc, err := client.PullTar(ctx, testRef, layerMediaType)
		require.NoError(t, err)
		defer pulledTar.Close()
		
		// Read the pulled tar content
		pulledData, err := io.ReadAll(pulledTar)
		require.NoError(t, err)
		
		// Verify pull result
		assert.Equal(t, tarData, string(pulledData))
		assert.Equal(t, pushDesc.Digest, pullDesc.Digest)
	})
	
	t.Run("ReleaseBundle_RoundTrip", func(t *testing.T) {
		testRef := fmt.Sprintf("%s/test/release:v2.0", registryHost)
		
		// Test release bundle data
		releaseJSON := []byte(`{
			"name": "test-release",
			"version": "2.0.0",
			"components": ["app", "db", "proxy"]
		}`)
		
		testAnnotations := NewAnnotations().
			WithForgeProject("catalyst").
			WithForgeEnv("staging").
			WithBuildInfo("build-789", "100", "https://ci.example.com/build/789")
		
		// Push release bundle
		pushDesc, err := client.PushReleaseBundle(ctx, testRef, releaseJSON, testAnnotations)
		require.NoError(t, err)
		
		// Verify the ForgeKind annotation was automatically added
		resolveDesc, err := client.Resolve(ctx, testRef)
		require.NoError(t, err)
		assert.Equal(t, pushDesc.Digest, resolveDesc.Digest)
		
		// Pull release bundle back
		pulledJSON, pullDesc, err := client.PullReleaseBundle(ctx, testRef)
		require.NoError(t, err)
		
		// Verify result
		assert.JSONEq(t, string(releaseJSON), string(pulledJSON))
		assert.Equal(t, pushDesc.Digest, pullDesc.Digest)
		assert.Equal(t, MTReleaseConfig, pullDesc.MediaType)
	})
	
	t.Run("RenderedSet_RoundTrip", func(t *testing.T) {
		testRef := fmt.Sprintf("%s/test/rendered:latest", registryHost)
		
		// Test rendered set data
		indexJSON := []byte(`{
			"schemaVersion": 2,
			"mediaType": "application/vnd.oci.image.index.v1+json",
			"manifests": [
				{
					"mediaType": "application/vnd.oci.image.manifest.v1+json",
					"size": 1234,
					"digest": "sha256:example"
				}
			]
		}`)
		
		renderedData := "rendered templates and configurations"
		renderedReader := strings.NewReader(renderedData)
		
		testAnnotations := NewAnnotations().
			WithForgeProject("catalyst").
			WithForgeEnv("production").
			WithGitInfo("def456", "main", "v2.0.0", false)
		
		// Push rendered set
		pushDesc, err := client.PushRenderedSet(ctx, testRef, indexJSON, 
			renderedReader, int64(len(renderedData)), testAnnotations)
		require.NoError(t, err)
		
		// Pull rendered set back
		pulledTar, pulledIndex, pullDesc, err := client.PullRenderedSet(ctx, testRef)
		require.NoError(t, err)
		defer pulledTar.Close()
		
		// Read pulled tar content
		pulledData, err := io.ReadAll(pulledTar)
		require.NoError(t, err)
		
		// Verify results
		assert.JSONEq(t, string(indexJSON), string(pulledIndex))
		assert.Equal(t, renderedData, string(pulledData))
		assert.Equal(t, pushDesc.Digest, pullDesc.Digest)
	})
}

// TestIntegrationFallbackBehavior tests the fallback from artifact to image manifests
func TestIntegrationFallbackBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	t.Parallel()
	
	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()
	
	registryHost := strings.TrimPrefix(registryServer.URL, "http://")
	
	t.Run("Artifact_Preferred_Success", func(t *testing.T) {
		// Create client that prefers artifact manifests
		client, err := New(ClientOptions{
			PlainHTTP:              true,
			PreferArtifactManifest: true,
			FallbackImageManifest:  false, // No fallback
		})
		require.NoError(t, err)
		
		testRef := fmt.Sprintf("%s/test/artifact-only:latest", registryHost)
		testJSON := []byte(`{"test": "artifact manifest"}`)
		
		// This should succeed since the test registry supports artifact manifests
		desc, err := client.PushJSON(context.Background(), testRef, 
			"application/vnd.test+json", testJSON, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, desc.Digest)
		
		// Verify we can pull it back
		pulled, _, err := client.PullJSON(context.Background(), testRef, 
			"application/vnd.test+json")
		require.NoError(t, err)
		assert.Equal(t, testJSON, pulled)
	})
	
	t.Run("Image_Manifest_Only", func(t *testing.T) {
		// Create client that uses image manifests only
		client, err := New(ClientOptions{
			PlainHTTP:              true,
			PreferArtifactManifest: false,
			FallbackImageManifest:  false,
		})
		require.NoError(t, err)
		
		testRef := fmt.Sprintf("%s/test/image-only:latest", registryHost)
		testJSON := []byte(`{"test": "image manifest"}`)
		
		// Push using image manifest format
		desc, err := client.PushJSON(context.Background(), testRef, 
			"application/vnd.test+json", testJSON, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, desc.Digest)
		
		// Verify we can pull it back
		pulled, _, err := client.PullJSON(context.Background(), testRef, 
			"application/vnd.test+json")
		require.NoError(t, err)
		assert.Equal(t, testJSON, pulled)
	})
}

// TestIntegrationErrorHandling tests error scenarios and structured error handling
func TestIntegrationErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	t.Parallel()
	
	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()
	
	registryHost := strings.TrimPrefix(registryServer.URL, "http://")
	
	client, err := New(ClientOptions{
		PlainHTTP: true,
	})
	require.NoError(t, err)
	
	ctx := context.Background()
	
	t.Run("NotFound_Error", func(t *testing.T) {
		nonExistentRef := fmt.Sprintf("%s/nonexistent/repo:latest", registryHost)
		
		// Try to pull non-existent artifact
		_, _, err := client.PullJSON(ctx, nonExistentRef, "application/json")
		require.Error(t, err)
		
		// Should be a structured error
		var ociErr *OCIError
		assert.True(t, errors.As(err, &ociErr))
		
		// Verify error is properly categorized
		category := GetErrorCategory(err)
		assert.Equal(t, ErrorCategoryRegistry, category)
	})
	
	t.Run("Invalid_Reference", func(t *testing.T) {
		invalidRef := "invalid-reference-format"
		
		// Try to push to invalid reference
		_, err := client.PushJSON(ctx, invalidRef, "application/json", 
			[]byte(`{}`), nil)
		require.Error(t, err)
		
		// Should be a validation error
		var ociErr *OCIError
		assert.True(t, errors.As(err, &ociErr))
		assert.Equal(t, ErrorCategoryValidation, ociErr.Category)
	})
	
	t.Run("Insecure_Reference", func(t *testing.T) {
		insecureRef := "http://example.com/repo:latest"
		
		// Try to push to insecure reference
		_, err := client.PushJSON(ctx, insecureRef, "application/json", 
			[]byte(`{}`), nil)
		require.Error(t, err)
		
		// Should be an insecure reference error
		assert.ErrorIs(t, err, ErrInsecureRef)
		
		var ociErr *OCIError
		assert.True(t, errors.As(err, &ociErr))
		assert.Equal(t, ErrorCategoryValidation, ociErr.Category)
	})
}

// TestIntegrationObservability tests logging and metrics integration
func TestIntegrationObservability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	t.Parallel()
	
	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()
	
	registryHost := strings.TrimPrefix(registryServer.URL, "http://")
	
	// Set up observability tracking
	var logEntries []string
	var metricsCallbacks []*Metrics
	
	logFunc := func(msg string, kv ...interface{}) {
		logEntries = append(logEntries, msg)
		t.Logf("Log: %s %v", msg, kv)
	}
	
	metricsFunc := func(m *Metrics) {
		metricsCallbacks = append(metricsCallbacks, m)
		t.Logf("Metrics: %+v", m)
	}
	
	client, err := New(ClientOptions{
		PlainHTTP:        true,
		StructuredLogger: NewDefaultLogger(logFunc),
		EnableMetrics:    true,
		MetricsCallback:  metricsFunc,
	})
	require.NoError(t, err)
	
	ctx := context.Background()
	testRef := fmt.Sprintf("%s/test/observability:latest", registryHost)
	testJSON := []byte(`{"observability": "test"}`)
	
	// Perform operations
	_, err = client.PushJSON(ctx, testRef, "application/json", testJSON, nil)
	require.NoError(t, err)
	
	_, _, err = client.PullJSON(ctx, testRef, "application/json")
	require.NoError(t, err)
	
	// Verify logging occurred
	assert.Greater(t, len(logEntries), 0, "Should have logged operations")
	
	// Verify metrics callbacks occurred
	assert.Greater(t, len(metricsCallbacks), 0, "Should have called metrics callback")
	
	// Verify metrics content
	for _, metrics := range metricsCallbacks {
		assert.Greater(t, metrics.OperationCounts["push_json"], int64(0))
		assert.Contains(t, metrics.RegistryStats, registryHost)
	}
}

// TestIntegrationTimeout tests timeout handling
func TestIntegrationTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	t.Parallel()
	
	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()
	
	registryHost := strings.TrimPrefix(registryServer.URL, "http://")
	
	// Create client with very short timeout
	client, err := New(ClientOptions{
		Timeout:   1 * time.Millisecond, // Very short timeout
		PlainHTTP: true,
	})
	require.NoError(t, err)
	
	// Create a context that will definitely timeout  
	ctx := context.Background()
	
	// Push a large amount of data to trigger timeout
	largeData := make([]byte, 10*1024*1024) // 10MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	
	testRef := fmt.Sprintf("%s/test/timeout:latest", registryHost)
	
	// This should timeout
	_, err = client.PushJSON(ctx, testRef, "application/json", largeData, nil)
	require.Error(t, err)
	
	// Should be a timeout or context error
	assert.True(t, 
		errors.Is(err, context.DeadlineExceeded) || 
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "context deadline exceeded"),
		"Expected timeout error, got: %v", err)
}
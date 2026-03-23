package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/input-output-hk/catalyst-forge/foundry/renderer/pkg/proto"
	project "github.com/input-output-hk/catalyst-forge/lib/schema/proto/generated/project"
)

// newTestRendererClient creates a new gRPC client for testing
func newTestRendererClient() (pb.RendererServiceClient, *grpc.ClientConn, error) {
	rendererURL := getEnv("RENDERER_URL", "localhost:8080")

	conn, err := grpc.Dial(rendererURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to renderer: %v", err)
	}

	client := pb.NewRendererServiceClient(conn)
	return client, conn, nil
}

// newTestContext creates a new context with timeout for testing
func newTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getKeys returns the keys from a map
func getKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestRendererIntegration(t *testing.T) {
	registryURL := getEnv("REGISTRY_URL", "localhost:5000")
	moduleName := getEnv("MODULE_NAME", "example-app")
	moduleVersion := getEnv("MODULE_VERSION", "v1.0.0")

	t.Logf("Starting integration test")
	t.Logf("Registry URL: %s", registryURL)
	t.Logf("Module: %s:%s", moduleName, moduleVersion)

	ctx, cancel := newTestContext()
	defer cancel()

	client, conn, err := newTestRendererClient()
	require.NoError(t, err)
	defer conn.Close()

	t.Run("HealthCheck", func(t *testing.T) {
		t.Log("Testing health check...")

		healthResp, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{})
		require.NoError(t, err, "Health check should succeed")

		assert.NotEmpty(t, healthResp.Status, "Health status should not be empty")
		assert.NotZero(t, healthResp.Timestamp, "Health timestamp should not be zero")

		t.Logf("Health check passed: %s at %d", healthResp.Status, healthResp.Timestamp)
	})

	t.Run("RenderManifests", func(t *testing.T) {
		t.Log("Testing manifest rendering...")

		bundle := &project.ModuleBundle{
			Env: "dev",
			Modules: map[string]*project.Module{
				moduleName: {
					Name:      moduleName,
					Registry:  registryURL,
					Version:   moduleVersion,
					Type:      "kcl",
					Namespace: "default",
					Instance:  "foundry-api",
					Values: []byte(`{
						"image": "nginx:1.20",
						"replicas": 2,
						"port": 80
					}`),
				},
			},
		}

		renderReq := &pb.RenderManifeststRequest{
			Bundle: bundle,
		}

		renderResp, err := client.RenderManifests(ctx, renderReq)
		require.NoError(t, err, "Render manifests should succeed")
		require.Empty(t, renderResp.Error, "Renderer should not return an error")
		require.NotEmpty(t, renderResp.Manifests, "Should render at least one manifest")

		t.Logf("Successfully rendered %d manifests", len(renderResp.Manifests))

		manifestKey := moduleName
		manifestYAML, exists := renderResp.Manifests[manifestKey]
		require.True(t, exists, "Expected manifest '%s' should exist. Available: %v", manifestKey, getKeys(renderResp.Manifests))

		t.Logf("Rendered manifest for '%s':", manifestKey)
		t.Logf("%s", string(manifestYAML))

		manifestStr := string(manifestYAML)
		expectedContent := []string{
			"apiVersion: apps/v1",
			"kind: Deployment",
			"name: foundry-api",
			"image: nginx:1.20",
			"replicas: 2",
			"containerPort: 80",
			"app.kubernetes.io/managed-by: forge",
		}

		for _, expected := range expectedContent {
			assert.Contains(t, manifestStr, expected, "Manifest should contain expected content: %s", expected)
		}
	})

	t.Run("CachingTest", func(t *testing.T) {
		t.Log("Testing caching with second render request...")

		bundle := &project.ModuleBundle{
			Env: "dev",
			Modules: map[string]*project.Module{
				moduleName: {
					Name:      moduleName,
					Registry:  registryURL,
					Version:   moduleVersion,
					Type:      "kcl",
					Namespace: "default",
					Instance:  "foundry-api",
					Values: []byte(`{
						"image": "nginx:1.20",
						"replicas": 2,
						"port": 80
					}`),
				},
			},
		}

		renderReq := &pb.RenderManifeststRequest{
			Bundle: bundle,
		}

		// First render
		renderResp1, err := client.RenderManifests(ctx, renderReq)
		require.NoError(t, err, "First render should succeed")
		require.Empty(t, renderResp1.Error, "First render should not return an error")

		// Second render
		renderResp2, err := client.RenderManifests(ctx, renderReq)
		require.NoError(t, err, "Second render should succeed")
		require.Empty(t, renderResp2.Error, "Second render should not return an error")

		manifestKey := moduleName
		manifestYAML1, exists1 := renderResp1.Manifests[manifestKey]
		require.True(t, exists1, "First render should have manifest '%s'", manifestKey)

		manifestYAML2, exists2 := renderResp2.Manifests[manifestKey]
		require.True(t, exists2, "Second render should have manifest '%s'", manifestKey)

		assert.Equal(t, string(manifestYAML1), string(manifestYAML2), "Second render should produce identical result")
	})
}

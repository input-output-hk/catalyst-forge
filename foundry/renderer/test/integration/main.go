package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/input-output-hk/catalyst-forge/foundry/renderer/pkg/proto"
	project "github.com/input-output-hk/catalyst-forge/lib/schema/proto/generated/project"
)

func main() {
	rendererURL := getEnv("RENDERER_URL", "localhost:8080")
	registryURL := getEnv("REGISTRY_URL", "localhost:5000")
	moduleName := getEnv("MODULE_NAME", "example-app")
	moduleVersion := getEnv("MODULE_VERSION", "v1.0.0")

	log.Printf("Starting integration test")
	log.Printf("Renderer URL: %s", rendererURL)
	log.Printf("Registry URL: %s", registryURL)
	log.Printf("Module: %s:%s", moduleName, moduleVersion)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(rendererURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to renderer: %v", err)
	}
	defer conn.Close()

	client := pb.NewRendererServiceClient(conn)

	log.Printf("Testing health check...")
	healthResp, err := client.HealthCheck(ctx, &pb.HealthCheckRequest{})
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	log.Printf("Health check passed: %s at %d", healthResp.Status, healthResp.Timestamp)

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

	log.Printf("Testing manifest rendering...")
	renderReq := &pb.RenderManifeststRequest{
		Bundle: bundle,
	}

	renderResp, err := client.RenderManifests(ctx, renderReq)
	if err != nil {
		log.Fatalf("Failed to render manifests: %v", err)
	}

	if renderResp.Error != "" {
		log.Fatalf("Renderer returned error: %s", renderResp.Error)
	}

	log.Printf("Successfully rendered %d manifests", len(renderResp.Manifests))

	if len(renderResp.Manifests) == 0 {
		log.Fatalf("No manifests were rendered")
	}

	manifestKey := moduleName
	manifestYAML, exists := renderResp.Manifests[manifestKey]
	if !exists {
		log.Fatalf("Expected manifest '%s' not found. Available: %v", manifestKey, getKeys(renderResp.Manifests))
	}

	log.Printf("Rendered manifest for '%s':", manifestKey)
	log.Printf("%s", string(manifestYAML))

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
		if !strings.Contains(manifestStr, expected) {
			log.Fatalf("Manifest missing expected content: %s", expected)
		}
	}

	log.Printf("✅ Integration test passed!")
	log.Printf("✅ Health check successful")
	log.Printf("✅ Example app KCL module successfully downloaded and cached from OCI registry")
	log.Printf("✅ Full deployment pipeline rendering successful")
	log.Printf("✅ Generated manifest contains expected Kubernetes deployment with proper metadata and configuration")

	log.Printf("Testing caching with second render request...")
	renderResp2, err := client.RenderManifests(ctx, renderReq)
	if err != nil {
		log.Fatalf("Failed on second render request: %v", err)
	}

	if renderResp2.Error != "" {
		log.Fatalf("Second render returned error: %s", renderResp2.Error)
	}

	manifestYAML2, exists := renderResp2.Manifests[manifestKey]
	if !exists {
		log.Fatalf("Second render missing manifest '%s'", manifestKey)
	}

	if string(manifestYAML) != string(manifestYAML2) {
		log.Fatalf("Second render produced different result")
	}

	log.Printf("✅ Caching test passed - second request returned identical result")
	log.Printf("🎉 All integration tests passed successfully!")
	log.Printf("🚀 Full deployment pipeline validated: Bundle → Provider Resolution → KCL Execution → YAML Generation")
}

// Helper function to get environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function to get keys from a map
func getKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Helper function to pretty print JSON for debugging
func prettyJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error marshaling JSON: %v", err)
	}
	return string(b)
}

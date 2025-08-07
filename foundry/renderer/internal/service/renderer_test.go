package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/foundry/renderer/pkg/proto"
	"github.com/input-output-hk/catalyst-forge/lib/deployment"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/proto/generated/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
)

func TestRendererService_HealthCheck(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*testing.T, *proto.HealthCheckResponse, error)
	}{
		{
			name: "health_check_success",
			validate: func(t *testing.T, resp *proto.HealthCheckResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "ok", resp.Status)
				assert.Greater(t, resp.Timestamp, int64(0))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock manifest generator store
			store := deployment.NewManifestGeneratorStore(map[deployment.Provider]func(*slog.Logger) (deployment.ManifestGenerator, error){})

			// Create service with noop logger for tests
			logger := testutils.NewNoopLogger()
			service := NewRendererService(store, logger)

			// Execute test
			req := &proto.HealthCheckRequest{}
			resp, err := service.HealthCheck(context.Background(), req)
			tt.validate(t, resp, err)
		})
	}
}

func TestRendererService_RenderManifests(t *testing.T) {
	tests := []struct {
		name     string
		bundle   *sp.ModuleBundle
		envData  []byte
		validate func(*testing.T, *proto.RenderManifestsResponse, error)
	}{
		{
			name:   "nil_bundle",
			bundle: nil,
			validate: func(t *testing.T, resp *proto.RenderManifestsResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Error)
				assert.Equal(t, "bundle is required", resp.Error)
			},
		},
		{
			name: "empty_bundle",
			bundle: &sp.ModuleBundle{
				Env:     "test",
				Modules: map[string]*sp.Module{},
			},
			validate: func(t *testing.T, resp *proto.RenderManifestsResponse, err error) {
				require.NoError(t, err)
				// Empty bundle should succeed but produce no manifests
				assert.Empty(t, resp.Error)
				assert.NotNil(t, resp.Manifests)
				assert.Len(t, resp.Manifests, 0)
			},
		},
		{
			name: "valid_bundle_unknown_provider",
			bundle: &sp.ModuleBundle{
				Env: "test",
				Modules: map[string]*sp.Module{
					"example-app": {
						Name:      "test",
						Type:      "unknown-provider",
						Instance:  "test-instance",
						Namespace: "default",
						Path:      "/test/path",
						Registry:  "registry.example.com",
						Version:   "v1.0.0",
						Values:    []byte(`{"key": "value"}`),
					},
				},
			},
			validate: func(t *testing.T, resp *proto.RenderManifestsResponse, err error) {
				require.NoError(t, err)
				// Should fail due to unknown provider
				assert.NotEmpty(t, resp.Error)
				assert.Contains(t, resp.Error, "unknown deployment module type")
			},
		},
		{
			name: "invalid_env_data",
			bundle: &sp.ModuleBundle{
				Env:     "test",
				Modules: map[string]*sp.Module{},
			},
			envData: []byte(`{invalid json`),
			validate: func(t *testing.T, resp *proto.RenderManifestsResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Error)
				assert.Contains(t, resp.Error, "failed to parse environment data")
			},
		},
		{
			name: "valid_env_data",
			bundle: &sp.ModuleBundle{
				Env:     "test",
				Modules: map[string]*sp.Module{},
			},
			envData: []byte(`{"environment": "test", "debug": true}`),
			validate: func(t *testing.T, resp *proto.RenderManifestsResponse, err error) {
				require.NoError(t, err)
				assert.Empty(t, resp.Error)
				assert.NotNil(t, resp.Manifests)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock manifest generator store
			store := deployment.NewManifestGeneratorStore(map[deployment.Provider]func(*slog.Logger) (deployment.ManifestGenerator, error){})

			// Create service with noop logger for tests
			logger := testutils.NewNoopLogger()
			service := NewRendererService(store, logger)

			// Execute test
			req := &proto.RenderManifeststRequest{
				Bundle:  tt.bundle,
				EnvData: tt.envData,
			}
			resp, err := service.RenderManifests(context.Background(), req)
			tt.validate(t, resp, err)
		})
	}
}

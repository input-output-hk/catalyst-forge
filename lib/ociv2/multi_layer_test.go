package ociv2

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackOptions_Validate(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name    string
		opts    PackOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_with_layers",
			opts: PackOptions{
				ArtifactType: "application/vnd.test+json",
				Layers: []LayerSpec{
					{
						MediaType: "application/vnd.test.layer+tar",
						Size:      1024,
						Reader:    bytes.NewReader([]byte("test")),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_with_config_only",
			opts: PackOptions{
				Config:          []byte("{}"),
				ConfigMediaType: "application/json",
			},
			wantErr: false,
		},
		{
			name: "invalid_no_layers_or_config",
			opts: PackOptions{
				ArtifactType: "application/vnd.test+json",
			},
			wantErr: true,
			errMsg:  "at least one layer or config is required",
		},
		{
			name: "invalid_config_without_media_type",
			opts: PackOptions{
				Config: []byte("{}"),
			},
			wantErr: true,
			errMsg:  "config media type is required",
		},
		{
			name: "invalid_layer_no_media_type",
			opts: PackOptions{
				Layers: []LayerSpec{
					{
						Size:   1024,
						Reader: bytes.NewReader([]byte("test")),
					},
				},
			},
			wantErr: true,
			errMsg:  "media type is required",
		},
		{
			name: "invalid_layer_negative_size",
			opts: PackOptions{
				Layers: []LayerSpec{
					{
						MediaType: "application/tar",
						Size:      -1,
						Reader:    bytes.NewReader([]byte("test")),
					},
				},
			},
			wantErr: true,
			errMsg:  "size cannot be negative",
		},
		{
			name: "invalid_layer_no_reader",
			opts: PackOptions{
				Layers: []LayerSpec{
					{
						MediaType: "application/tar",
						Size:      1024,
					},
				},
			},
			wantErr: true,
			errMsg:  "reader is required",
		},
		{
			name: "invalid_artifact_manifest_no_type",
			opts: PackOptions{
				PreferArtifactManifest: true,
				Layers: []LayerSpec{
					{
						MediaType: "application/tar",
						Size:      1024,
						Reader:    bytes.NewReader([]byte("test")),
					},
				},
			},
			wantErr: true,
			errMsg:  "artifact type is required",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPullResult_Helpers(t *testing.T) {
	t.Parallel()
	
	result := &PullResult{
		Config:          []byte("{}"),
		ConfigMediaType: "application/json",
		Layers: []PulledLayer{
			{
				MediaType: "application/tar",
				Size:      1024,
				Digest:    "sha256:abc123",
			},
			{
				MediaType: "application/gzip",
				Size:      2048,
				Digest:    "sha256:def456",
			},
		},
	}
	
	t.Run("HasConfig", func(t *testing.T) {
		assert.True(t, result.HasConfig())
		
		emptyResult := &PullResult{}
		assert.False(t, emptyResult.HasConfig())
	})
	
	t.Run("LayerCount", func(t *testing.T) {
		assert.Equal(t, 2, result.LayerCount())
	})
	
	t.Run("GetLayer", func(t *testing.T) {
		layer, err := result.GetLayer(0)
		require.NoError(t, err)
		assert.Equal(t, "application/tar", layer.MediaType)
		
		layer, err = result.GetLayer(1)
		require.NoError(t, err)
		assert.Equal(t, "application/gzip", layer.MediaType)
		
		_, err = result.GetLayer(2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "out of bounds")
		
		_, err = result.GetLayer(-1)
		assert.Error(t, err)
	})
	
	t.Run("GetLayerByMediaType", func(t *testing.T) {
		layer, err := result.GetLayerByMediaType("application/tar")
		require.NoError(t, err)
		assert.Equal(t, "sha256:abc123", layer.Digest)
		
		layer, err = result.GetLayerByMediaType("application/gzip")
		require.NoError(t, err)
		assert.Equal(t, "sha256:def456", layer.Digest)
		
		_, err = result.GetLayerByMediaType("application/unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no layer found")
	})
}

func TestIntegrationPushPullArtifact(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	t.Parallel()
	
	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()
	
	registryHost := strings.TrimPrefix(registryServer.URL, "http://")
	
	// Create client
	client, err := New(ClientOptions{
		PlainHTTP:              true,
		PreferArtifactManifest: true,
		FallbackImageManifest:  true,
	})
	require.NoError(t, err)
	
	ctx := context.Background()
	
	t.Run("SingleLayer", func(t *testing.T) {
		ref := fmt.Sprintf("%s/test/single-layer:v1", registryHost)
		
		// Prepare layer
		layerData := []byte("This is test layer content")
		
		opts := PackOptions{
			ArtifactType: "application/vnd.test.artifact+tar",
			ManifestAnnotations: map[string]string{
				"test.annotation": "value1",
				"test.version":    "1.0.0",
			},
			Layers: []LayerSpec{
				{
					MediaType: "application/vnd.test.layer+tar",
					Title:     "Test Layer",
					Size:      int64(len(layerData)),
					Reader:    bytes.NewReader(layerData),
					Annotations: map[string]string{
						"layer.type": "primary",
					},
				},
			},
		}
		
		// Push artifact
		pushDesc, err := client.PushArtifact(ctx, ref, opts)
		require.NoError(t, err)
		assert.NotEmpty(t, pushDesc.Digest)
		assert.Contains(t, pushDesc.Ref, "@sha256:")
		
		// Pull artifact back
		result, err := client.PullArtifact(ctx, pushDesc.Ref)
		require.NoError(t, err)
		
		// Verify result
		assert.Equal(t, 1, result.LayerCount())
		assert.Equal(t, "value1", result.ManifestAnn["test.annotation"])
		assert.Equal(t, "1.0.0", result.ManifestAnn["test.version"])
		
		// Verify layer
		layer, err := result.GetLayer(0)
		require.NoError(t, err)
		assert.Equal(t, "application/vnd.test.layer+tar", layer.MediaType)
		assert.Equal(t, int64(len(layerData)), layer.Size)
		
		// Read layer content
		rc, err := layer.Open()
		require.NoError(t, err)
		defer rc.Close()
		
		content, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, layerData, content)
	})
	
	t.Run("MultiLayer", func(t *testing.T) {
		ref := fmt.Sprintf("%s/test/multi-layer:v1", registryHost)
		
		// Prepare layers
		layer1Data := []byte("Layer 1 content")
		layer2Data := []byte("Layer 2 content with more data")
		configData := []byte(`{"config": "test", "version": "1.0"}`)
		
		opts := PackOptions{
			ArtifactType: "application/vnd.test.multi+json",
			Config:       configData,
			ConfigMediaType: "application/vnd.test.config+json",
			ManifestAnnotations: map[string]string{
				"multi.test": "true",
			},
			Layers: []LayerSpec{
				{
					MediaType: "application/vnd.test.layer1+tar",
					Title:     "First Layer",
					Size:      int64(len(layer1Data)),
					Reader:    bytes.NewReader(layer1Data),
				},
				{
					MediaType: "application/vnd.test.layer2+json",
					Title:     "Second Layer",
					Size:      int64(len(layer2Data)),
					Reader:    bytes.NewReader(layer2Data),
				},
			},
		}
		
		// Push artifact
		pushDesc, err := client.PushArtifact(ctx, ref, opts)
		require.NoError(t, err)
		
		// Pull artifact back
		result, err := client.PullArtifact(ctx, pushDesc.Ref)
		require.NoError(t, err)
		
		// Verify result
		assert.Equal(t, 2, result.LayerCount())
		assert.True(t, result.HasConfig())
		assert.Equal(t, "application/vnd.test.config+json", result.ConfigMediaType)
		assert.Equal(t, configData, result.Config)
		
		// Verify layers
		layer1, err := result.GetLayerByMediaType("application/vnd.test.layer1+tar")
		require.NoError(t, err)
		assert.Equal(t, int64(len(layer1Data)), layer1.Size)
		
		layer2, err := result.GetLayerByMediaType("application/vnd.test.layer2+json")
		require.NoError(t, err)
		assert.Equal(t, int64(len(layer2Data)), layer2.Size)
		
		// Read layer contents
		rc1, err := layer1.Open()
		require.NoError(t, err)
		defer rc1.Close()
		content1, _ := io.ReadAll(rc1)
		assert.Equal(t, layer1Data, content1)
		
		rc2, err := layer2.Open()
		require.NoError(t, err)
		defer rc2.Close()
		content2, _ := io.ReadAll(rc2)
		assert.Equal(t, layer2Data, content2)
	})
	
	t.Run("ImageManifestFallback", func(t *testing.T) {
		ref := fmt.Sprintf("%s/test/fallback:v1", registryHost)
		
		layerData := []byte("Fallback test content")
		
		opts := PackOptions{
			// Don't set artifact type to force image manifest
			PreferArtifactManifest: false,
			FallbackImageManifest:  true,
			Config:                 []byte("{}"),
			ConfigMediaType:        "application/json",
			Layers: []LayerSpec{
				{
					MediaType: "application/tar",
					Size:      int64(len(layerData)),
					Reader:    bytes.NewReader(layerData),
				},
			},
		}
		
		// Push artifact (will use image manifest)
		pushDesc, err := client.PushArtifact(ctx, ref, opts)
		require.NoError(t, err)
		
		// Pull artifact back
		result, err := client.PullArtifact(ctx, pushDesc.Ref)
		require.NoError(t, err)
		
		// Should still work
		assert.Equal(t, 1, result.LayerCount())
		assert.True(t, result.HasConfig())
	})
}

func TestIntegrationPushArtifactErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	t.Parallel()
	
	client, err := New(ClientOptions{})
	require.NoError(t, err)
	
	ctx := context.Background()
	
	t.Run("InvalidReference", func(t *testing.T) {
		opts := PackOptions{
			Layers: []LayerSpec{
				{
					MediaType: "application/tar",
					Size:      10,
					Reader:    bytes.NewReader([]byte("test")),
				},
			},
		}
		
		_, err := client.PushArtifact(ctx, "invalid ref", opts)
		assert.Error(t, err)
		
		var ociErr *OCIError
		if assert.ErrorAs(t, err, &ociErr) {
			assert.Equal(t, ErrorCategoryValidation, ociErr.Category)
		}
	})
	
	t.Run("InvalidOptions", func(t *testing.T) {
		invalidOpts := PackOptions{} // No layers or config
		
		_, err := client.PushArtifact(ctx, "registry.example.com/test:v1", invalidOpts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one layer or config")
	})
}
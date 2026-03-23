package oci

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

func TestOrasClient_Pull(t *testing.T) {
	tests := []struct {
		name     string
		imageURL string
		destPath string
		validate func(*testing.T, error)
	}{
		{
			name:     "empty_image_url",
			imageURL: "",
			destPath: "/tmp/test",
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Equal(t, "image URL cannot be empty", err.Error())
			},
		},
		{
			name:     "empty_dest_path",
			imageURL: "ghcr.io/test/image:latest",
			destPath: "",
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Equal(t, "destination path cannot be empty", err.Error())
			},
		},
		{
			name:     "nil_store",
			imageURL: "ghcr.io/test/image:latest",
			destPath: "/tmp/test",
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "no store configured")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *OrasClient
			if tt.name == "nil_store" {
				// Create client with nil store
				client = &OrasClient{
					ctx:                 context.Background(),
					store:               nil,
					destFS:              billy.NewInMemoryFs(),
					maxManifestSize:     10 * 1024 * 1024,
					maxTotalExtractSize: 1024 * 1024 * 1024,
					maxFileSize:         100 * 1024 * 1024,
					maxFileCount:        10000,
				}
			} else {
				var err error
				client, err = New(WithDestFS(billy.NewInMemoryFs()))
				require.NoError(t, err)
			}

			err := client.Pull(tt.imageURL, tt.destPath)
			tt.validate(t, err)
		})
	}
}

func TestOrasClient_IsTarLayer(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		validate  func(*testing.T, bool)
	}{
		{
			name:      "oci_tar_layer",
			mediaType: "application/vnd.oci.image.layer.v1.tar",
			validate: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:      "oci_tar_gzip_layer",
			mediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
			validate: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:      "octet_stream",
			mediaType: "application/octet-stream",
			validate: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:      "oci_config",
			mediaType: "application/vnd.oci.image.config.v1+json",
			validate: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
		{
			name:      "json",
			mediaType: "application/json",
			validate: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
		{
			name:      "empty_media_type",
			mediaType: "",
			validate: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New()
			require.NoError(t, err)
			result := client.isTarLayer(tt.mediaType)
			tt.validate(t, result)
		})
	}
}

func TestOrasClient_IsSecurePath(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		destDir  string
		validate func(*testing.T, bool)
	}{
		{
			name:    "valid_file_in_dest",
			target:  "/tmp/test/file.txt",
			destDir: "/tmp/test",
			validate: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:    "valid_nested_file",
			target:  "/tmp/test/subdir/file.txt",
			destDir: "/tmp/test",
			validate: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:    "exact_dest_dir",
			target:  "/tmp/test",
			destDir: "/tmp/test",
			validate: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
		{
			name:    "file_outside_dest",
			target:  "/tmp/other/file.txt",
			destDir: "/tmp/test",
			validate: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
		{
			name:    "system_file_traversal",
			target:  "/etc/passwd",
			destDir: "/tmp/test",
			validate: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
		{
			name:    "relative_path_traversal",
			target:  "../../../etc/passwd",
			destDir: "/tmp/test",
			validate: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
		{
			name:    "complex_path_traversal",
			target:  "/tmp/test/../../../etc/passwd",
			destDir: "/tmp/test",
			validate: func(t *testing.T, result bool) {
				assert.False(t, result)
			},
		},
		{
			name:    "dot_notation_valid",
			target:  "/tmp/test/./file.txt",
			destDir: "/tmp/test",
			validate: func(t *testing.T, result bool) {
				assert.True(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New()
			require.NoError(t, err)
			result := client.isSecurePath(tt.target, tt.destDir)
			tt.validate(t, result)
		})
	}
}

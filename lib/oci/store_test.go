package oci

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

func TestNewFilesystemStore(t *testing.T) {
	tests := []struct {
		name     string
		root     string
		validate func(*testing.T, *FilesystemStore, error)
	}{
		{
			name: "valid_root_path",
			root: "/test-store",
			validate: func(t *testing.T, store *FilesystemStore, err error) {
				require.NoError(t, err)
				assert.NotNil(t, store)
				assert.Equal(t, "/test-store", store.root)
				assert.NotNil(t, store.fs)
				assert.NotNil(t, store.manifests)
				assert.NotNil(t, store.blobs)
			},
		},
		{
			name: "nested_root_path",
			root: "/parent/child/store",
			validate: func(t *testing.T, store *FilesystemStore, err error) {
				require.NoError(t, err)
				assert.NotNil(t, store)
				assert.Equal(t, "/parent/child/store", store.root)
			},
		},
		{
			name: "root_path_with_trailing_slash",
			root: "/test-store/",
			validate: func(t *testing.T, store *FilesystemStore, err error) {
				require.NoError(t, err)
				assert.NotNil(t, store)
				assert.Equal(t, "/test-store/", store.root)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			store, err := NewFilesystemStore(fs, tt.root)
			tt.validate(t, store, err)

			// Verify directory structure was created
			if store != nil {
				blobsDir := store.root + "/blobs/sha256"
				exists, dirErr := fs.Exists(blobsDir)
				require.NoError(t, dirErr)
				assert.True(t, exists, "blobs directory should be created")
			}
		})
	}
}

func TestFilesystemStore_Push(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		descriptor ocispec.Descriptor
		validate   func(*testing.T, error)
	}{
		{
			name:    "valid_content",
			content: "hello world",
			descriptor: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				Digest:    digest.FromString("hello world"),
				Size:      int64(len("hello world")),
			},
			validate: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "empty_content",
			content: "",
			descriptor: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				Digest:    digest.FromString(""),
				Size:      0,
			},
			validate: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "large_content",
			content: strings.Repeat("test data ", 1000),
			descriptor: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				Digest:    digest.FromString(strings.Repeat("test data ", 1000)),
				Size:      int64(len(strings.Repeat("test data ", 1000))),
			},
			validate: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "digest_mismatch",
			content: "hello world",
			descriptor: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				Digest:    digest.FromString("different content"),
				Size:      int64(len("hello world")),
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "digest mismatch")
			},
		},
		{
			name:    "size_mismatch",
			content: "hello world",
			descriptor: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				Digest:    digest.FromString("hello world"),
				Size:      999, // Wrong size
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "size mismatch")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			store, err := NewFilesystemStore(fs, "/test-store")
			require.NoError(t, err)

			ctx := context.Background()
			err = store.Push(ctx, tt.descriptor, strings.NewReader(tt.content))
			tt.validate(t, err)
		})
	}
}

func TestFilesystemStore_Fetch(t *testing.T) {
	tests := []struct {
		name         string
		setupContent string
		descriptor   ocispec.Descriptor
		validate     func(*testing.T, []byte, error)
	}{
		{
			name:         "existing_blob",
			setupContent: "test content",
			descriptor: ocispec.Descriptor{
				Digest: digest.FromString("test content"),
			},
			validate: func(t *testing.T, content []byte, err error) {
				require.NoError(t, err)
				assert.Equal(t, "test content", string(content))
			},
		},
		{
			name:         "empty_blob",
			setupContent: "",
			descriptor: ocispec.Descriptor{
				Digest: digest.FromString(""),
			},
			validate: func(t *testing.T, content []byte, err error) {
				require.NoError(t, err)
				assert.Equal(t, "", string(content))
			},
		},
		{
			name:         "nonexistent_blob",
			setupContent: "",
			descriptor: ocispec.Descriptor{
				Digest: "sha256:nonexistent",
			},
			validate: func(t *testing.T, content []byte, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "blob sha256:nonexistent")
				assert.Nil(t, content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			store, err := NewFilesystemStore(fs, "/test-store")
			require.NoError(t, err)

			ctx := context.Background()

			// Setup: Push content if provided
			if tt.setupContent != "" || tt.name == "empty_blob" {
				setupDesc := ocispec.Descriptor{
					MediaType: "application/vnd.oci.image.layer.v1.tar",
					Digest:    digest.FromString(tt.setupContent),
					Size:      int64(len(tt.setupContent)),
				}
				err = store.Push(ctx, setupDesc, strings.NewReader(tt.setupContent))
				require.NoError(t, err)
			}

			// Test: Fetch content
			reader, err := store.Fetch(ctx, tt.descriptor)
			var content []byte
			if err == nil {
				defer reader.Close()
				content, err = io.ReadAll(reader)
			}
			tt.validate(t, content, err)
		})
	}
}

func TestFilesystemStore_Exists(t *testing.T) {
	tests := []struct {
		name         string
		setupContent string
		descriptor   ocispec.Descriptor
		validate     func(*testing.T, bool, error)
	}{
		{
			name:         "existing_blob",
			setupContent: "test content",
			descriptor: ocispec.Descriptor{
				Digest: digest.FromString("test content"),
			},
			validate: func(t *testing.T, exists bool, err error) {
				require.NoError(t, err)
				assert.True(t, exists)
			},
		},
		{
			name:         "nonexistent_blob",
			setupContent: "",
			descriptor: ocispec.Descriptor{
				Digest: "sha256:nonexistent",
			},
			validate: func(t *testing.T, exists bool, err error) {
				require.NoError(t, err)
				assert.False(t, exists)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			store, err := NewFilesystemStore(fs, "/test-store")
			require.NoError(t, err)

			ctx := context.Background()

			// Setup: Push content if provided
			if tt.setupContent != "" {
				setupDesc := ocispec.Descriptor{
					MediaType: "application/vnd.oci.image.layer.v1.tar",
					Digest:    digest.FromString(tt.setupContent),
					Size:      int64(len(tt.setupContent)),
				}
				err = store.Push(ctx, setupDesc, strings.NewReader(tt.setupContent))
				require.NoError(t, err)
			}

			// Test: Check existence
			exists, err := store.Exists(ctx, tt.descriptor)
			tt.validate(t, exists, err)
		})
	}
}

func TestFilesystemStore_TagAndResolve(t *testing.T) {
	tests := []struct {
		name       string
		reference  string
		descriptor ocispec.Descriptor
		validate   func(*testing.T, ocispec.Descriptor, error)
	}{
		{
			name:      "valid_reference",
			reference: "latest",
			descriptor: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    "sha256:abc123",
				Size:      1234,
			},
			validate: func(t *testing.T, resolved ocispec.Descriptor, err error) {
				require.NoError(t, err)
				assert.Equal(t, "application/vnd.oci.image.manifest.v1+json", resolved.MediaType)
				assert.Equal(t, digest.Digest("sha256:abc123"), resolved.Digest)
				assert.Equal(t, int64(1234), resolved.Size)
			},
		},
		{
			name:      "version_reference",
			reference: "v1.2.3",
			descriptor: ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    "sha256:def456",
				Size:      5678,
			},
			validate: func(t *testing.T, resolved ocispec.Descriptor, err error) {
				require.NoError(t, err)
				assert.Equal(t, digest.Digest("sha256:def456"), resolved.Digest)
			},
		},
		{
			name:       "nonexistent_reference",
			reference:  "nonexistent",
			descriptor: ocispec.Descriptor{},
			validate: func(t *testing.T, resolved ocispec.Descriptor, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "reference nonexistent")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			store, err := NewFilesystemStore(fs, "/test-store")
			require.NoError(t, err)

			ctx := context.Background()

			// Setup: Tag descriptor if provided
			if tt.descriptor.Digest != "" {
				err = store.Tag(ctx, tt.descriptor, tt.reference)
				require.NoError(t, err)
			}

			// Test: Resolve reference
			resolved, err := store.Resolve(ctx, tt.reference)
			tt.validate(t, resolved, err)
		})
	}
}

func TestFilesystemStore_Close(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*testing.T, error)
	}{
		{
			name: "successful_close",
			validate: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			store, err := NewFilesystemStore(fs, "/test-store")
			require.NoError(t, err)

			err = store.Close()
			tt.validate(t, err)

			// Verify maps are cleared
			assert.Nil(t, store.manifests)
			assert.Nil(t, store.blobs)
		})
	}
}

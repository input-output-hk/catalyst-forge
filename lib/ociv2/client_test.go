package ociv2

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    ClientOptions
		wantErr bool
	}{
		{
			name:    "ok/default_options",
			opts:    ClientOptions{},
			wantErr: false,
		},
		{
			name: "ok/with_timeout",
			opts: ClientOptions{
				Timeout: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "ok/with_cosign_enabled",
			opts: ClientOptions{
				Cosign: CosignOpts{
					Enable: true,
				},
			},
			wantErr: false,
		},
		{
			name: "ok/with_plain_http",
			opts: ClientOptions{
				PlainHTTP: true,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range var
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, err := New(tc.opts)
			if tc.wantErr {
				require.Error(t, err, "expected error for opts=%+v", tc.opts)
				return
			}
			require.NoError(t, err, "unexpected error for opts=%+v", tc.opts)
			assert.NotNil(t, client, "client should not be nil")
		})
	}
}

func TestAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("ok/creation", func(t *testing.T) {
		t.Parallel()
		ann := NewAnnotations()
		require.NotNil(t, ann, "NewAnnotations() should not return nil")
		assert.NotEmpty(t, ann[AnnCreated], "created timestamp should be set")
	})

	t.Run("ok/builders", func(t *testing.T) {
		t.Parallel()
		ann := NewAnnotations()
		ann = ann.WithSource("https://github.com/example/repo", "abc123")
		assert.Equal(t, "https://github.com/example/repo", ann[AnnSourceRepo], "source repo mismatch")
		assert.Equal(t, "abc123", ann[AnnSourceRev], "source revision mismatch")

		ann = ann.WithForgeKind("release")
		assert.Equal(t, "release", ann[AnnForgeKind], "forge kind mismatch")

		ann = ann.WithForgeProject("test-project")
		assert.Equal(t, "test-project", ann[AnnForgeProject], "forge project mismatch")
	})

	t.Run("ok/merge", func(t *testing.T) {
		t.Parallel()
		ann := NewAnnotations().
			WithForgeKind("release").
			WithForgeProject("test-project")
		
		ann2 := Annotations{
			"custom.key": "custom.value",
		}
		merged := ann.Merge(ann2)
		assert.Equal(t, "custom.value", merged["custom.key"], "merge should include new annotation")
		assert.Equal(t, "release", merged[AnnForgeKind], "merge should preserve existing annotations")
	})
}

func TestDescriptor(t *testing.T) {
	t.Parallel()

	desc := Descriptor{
		Ref:       "oci://registry.example.com/test@sha256:abc123",
		Digest:    "sha256:abc123",
		Size:      1024,
		MediaType: MTReleaseConfig,
		Annotations: Annotations{
			AnnForgeKind: "release",
		},
	}

	// Verify the descriptor fields
	assert.Equal(t, "oci://registry.example.com/test@sha256:abc123", desc.Ref, "descriptor ref mismatch")
	assert.Equal(t, "sha256:abc123", desc.Digest, "descriptor digest mismatch")
	assert.Equal(t, int64(1024), desc.Size, "descriptor size mismatch")
	assert.Equal(t, MTReleaseConfig, desc.MediaType, "descriptor media type mismatch")
	assert.Equal(t, "release", desc.Annotations[AnnForgeKind], "descriptor annotation mismatch")
}

func TestAuthProviders(t *testing.T) {
	t.Parallel()

	t.Run("ok/default_auth", func(t *testing.T) {
		t.Parallel()
		auth := &DefaultAuth{}
		authenticator, err := auth.Authenticator("registry.example.com")
		require.NoError(t, err, "DefaultAuth.Authenticator() should not error")
		assert.NotNil(t, authenticator, "DefaultAuth.Authenticator() should not return nil")
	})


	t.Run("ok/static_auth", func(t *testing.T) {
		t.Parallel()
		staticAuth := &StaticAuth{
			Username: "user",
			Password: "pass",
		}
		authenticator, err := staticAuth.Authenticator("registry.example.com")
		require.NoError(t, err, "StaticAuth.Authenticator() should not error")
		assert.NotNil(t, authenticator, "StaticAuth.Authenticator() should not return nil")
	})


	t.Run("ok/github_auth", func(t *testing.T) {
		t.Parallel()
		ghAuth := &GitHubAuth{
			Token: "ghp_test",
		}
		authenticator, err := ghAuth.Authenticator("ghcr.io")
		require.NoError(t, err, "GitHubAuth.Authenticator() should not error")
		assert.NotNil(t, authenticator, "GitHubAuth.Authenticator() should not return nil")
	})
}

func TestClientResolveValidation(t *testing.T) {
	t.Parallel()

	client, err := New(ClientOptions{
		Timeout: 1 * time.Second,
	})
	require.NoError(t, err, "Failed to create client")

	ctx := context.Background()

	t.Run("error/insecure_reference", func(t *testing.T) {
		_, err := client.Resolve(ctx, "http://insecure.com/image")
		assert.Error(t, err, "Resolve() should fail for insecure reference")
	})

	t.Run("error/unavailable_registry", func(t *testing.T) {
		// Test that operations fail gracefully when no registry is available
		// These should fail with network/auth errors, not panic
		_, err := client.Head(ctx, "oci://localhost:5000/test:latest")
		assert.Error(t, err, "Head() should fail when registry is not available")
	})
}
package oci

import (
	"context"
	"fmt"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCosignVerifier(t *testing.T) {
	tests := []struct {
		name     string
		opts     CosignOptions
		validate func(*testing.T, *CosignVerifier, error)
	}{
		{
			name: "valid_required_options",
			opts: CosignOptions{
				OIDCIssuer:  "https://token.actions.githubusercontent.com",
				OIDCSubject: "https://github.com/example/repo/.github/workflows/build.yml@refs/heads/main",
			},
			validate: func(t *testing.T, verifier *CosignVerifier, err error) {
				require.NoError(t, err)
				assert.NotNil(t, verifier)
				assert.Equal(t, "https://token.actions.githubusercontent.com", verifier.oidcIssuer)
				assert.Equal(t, "https://github.com/example/repo/.github/workflows/build.yml@refs/heads/main", verifier.oidcSubject)
				assert.Equal(t, "https://rekor.sigstore.dev", verifier.rekorURL) // default
				assert.False(t, verifier.skipTLog)                               // default
			},
		},
		{
			name: "valid_all_options",
			opts: CosignOptions{
				OIDCIssuer:  "https://token.actions.githubusercontent.com",
				OIDCSubject: "https://github.com/example/repo/.github/workflows/build.yml@refs/heads/main",
				RekorURL:    "https://custom.rekor.example.com",
				SkipTLog:    true,
			},
			validate: func(t *testing.T, verifier *CosignVerifier, err error) {
				require.NoError(t, err)
				assert.NotNil(t, verifier)
				assert.Equal(t, "https://token.actions.githubusercontent.com", verifier.oidcIssuer)
				assert.Equal(t, "https://github.com/example/repo/.github/workflows/build.yml@refs/heads/main", verifier.oidcSubject)
				assert.Equal(t, "https://custom.rekor.example.com", verifier.rekorURL)
				assert.True(t, verifier.skipTLog)
			},
		},
		{
			name: "missing_oidc_issuer",
			opts: CosignOptions{
				OIDCSubject: "https://github.com/example/repo/.github/workflows/build.yml@refs/heads/main",
			},
			validate: func(t *testing.T, verifier *CosignVerifier, err error) {
				assert.Error(t, err)
				assert.Equal(t, "OIDC issuer cannot be empty", err.Error())
				assert.Nil(t, verifier)
			},
		},
		{
			name: "missing_oidc_subject",
			opts: CosignOptions{
				OIDCIssuer: "https://token.actions.githubusercontent.com",
			},
			validate: func(t *testing.T, verifier *CosignVerifier, err error) {
				assert.Error(t, err)
				assert.Equal(t, "OIDC subject cannot be empty", err.Error())
				assert.Nil(t, verifier)
			},
		},
		{
			name: "empty_all_fields",
			opts: CosignOptions{},
			validate: func(t *testing.T, verifier *CosignVerifier, err error) {
				assert.Error(t, err)
				assert.Equal(t, "OIDC issuer cannot be empty", err.Error())
				assert.Nil(t, verifier)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier, err := NewCosignVerifier(tt.opts)
			tt.validate(t, verifier, err)
		})
	}
}

func TestCosignVerifier_Verify_InvalidImageRef(t *testing.T) {
	tests := []struct {
		name     string
		imageRef string
		validate func(*testing.T, error)
	}{
		{
			name:     "empty_image_ref",
			imageRef: "",
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to parse image reference")
			},
		},
		{
			name:     "invalid_image_ref",
			imageRef: "not-a-valid-image-ref::",
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to parse image reference")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier, err := NewCosignVerifier(CosignOptions{
				OIDCIssuer:  "https://token.actions.githubusercontent.com",
				OIDCSubject: "https://github.com/example/repo/.github/workflows/build.yml@refs/heads/main",
			})
			require.NoError(t, err)

			manifest := ocispec.Descriptor{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    "sha256:abc123",
				Size:      1234,
			}

			err = verifier.Verify(context.Background(), tt.imageRef, manifest)
			tt.validate(t, err)
		})
	}
}

// MockVerifier implements Verifier for testing
type MockVerifier struct {
	shouldError bool
	errorMsg    string
}

func (m *MockVerifier) Verify(ctx context.Context, imageRef string, manifest ocispec.Descriptor) error {
	if m.shouldError {
		return assert.AnError
	}
	if m.errorMsg != "" {
		return fmt.Errorf("%s", m.errorMsg)
	}
	return nil
}

func TestClientVerificationOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     []Option
		expectError bool
		validate    func(*testing.T, *OrasClient, error)
	}{
		{
			name: "with_cosign_verification",
			options: []Option{
				WithCosignVerification(
					"https://token.actions.githubusercontent.com",
					"https://github.com/example/repo/.github/workflows/build.yml@refs/heads/main",
				),
			},
			expectError: false,
			validate: func(t *testing.T, client *OrasClient, err error) {
				require.NoError(t, err)
				assert.NotNil(t, client.verifier)
			},
		},
		{
			name: "with_cosign_verification_advanced",
			options: []Option{
				WithCosignVerificationAdvanced(CosignOptions{
					OIDCIssuer:  "https://token.actions.githubusercontent.com",
					OIDCSubject: "https://github.com/example/repo/.github/workflows/build.yml@refs/heads/main",
					RekorURL:    "https://custom.rekor.example.com",
					SkipTLog:    true,
				}),
			},
			expectError: false,
			validate: func(t *testing.T, client *OrasClient, err error) {
				require.NoError(t, err)
				assert.NotNil(t, client.verifier)
			},
		},
		{
			name: "with_custom_verifier",
			options: []Option{
				WithCustomVerifier(&MockVerifier{}),
			},
			expectError: false,
			validate: func(t *testing.T, client *OrasClient, err error) {
				require.NoError(t, err)
				assert.NotNil(t, client.verifier)
				assert.IsType(t, &MockVerifier{}, client.verifier)
			},
		},
		{
			name:        "no_verification",
			options:     []Option{},
			expectError: false,
			validate: func(t *testing.T, client *OrasClient, err error) {
				require.NoError(t, err)
				assert.Nil(t, client.verifier)
			},
		},
		{
			name: "invalid_cosign_options",
			options: []Option{
				WithCosignVerification("", ""), // Invalid - both empty
			},
			expectError: true,
			validate: func(t *testing.T, client *OrasClient, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to apply option")
				assert.Nil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.options...)
			tt.validate(t, client, err)
		})
	}
}

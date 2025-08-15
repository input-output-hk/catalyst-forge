package registry

import (
	"errors"
	"net/http"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/ociv2/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectRegistryType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hostname string
		want     internal.RegistryType
	}{
		{
			name:     "ok/docker_hub",
			hostname: "docker.io",
			want:     internal.RegistryTypeDockerHub,
		},
		{
			name:     "ok/docker_hub_registry",
			hostname: "registry-1.docker.io",
			want:     internal.RegistryTypeDockerHub,
		},
		{
			name:     "ok/ecr_us_east",
			hostname: "123456789012.dkr.ecr.us-east-1.amazonaws.com",
			want:     internal.RegistryTypeECR,
		},
		{
			name:     "ok/ecr_eu_west",
			hostname: "123456789012.dkr.ecr.eu-west-1.amazonaws.com",
			want:     internal.RegistryTypeECR,
		},
		{
			name:     "ok/ghcr",
			hostname: "ghcr.io",
			want:     internal.RegistryTypeGHCR,
		},
		{
			name:     "ok/gcr",
			hostname: "gcr.io",
			want:     internal.RegistryTypeGCR,
		},
		{
			name:     "ok/gcr_regional",
			hostname: "us.gcr.io",
			want:     internal.RegistryTypeGCR,
		},
		{
			name:     "ok/acr",
			hostname: "myregistry.azurecr.io",
			want:     internal.RegistryTypeACR,
		},
		{
			name:     "ok/quay",
			hostname: "quay.io",
			want:     internal.RegistryTypeQuay,
		},
		{
			name:     "ok/generic_localhost",
			hostname: "localhost:5000",
			want:     internal.RegistryTypeGeneric,
		},
		{
			name:     "ok/generic_custom",
			hostname: "registry.example.com",
			want:     internal.RegistryTypeGeneric,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range var
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := internal.DetectRegistryType(tc.hostname)
			assert.Equal(t, tc.want, got, "registry type detection failed for hostname=%s", tc.hostname)
		})
	}
}

func TestShouldFallbackToImageManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		err          error
		registryType internal.RegistryType
		want         bool
	}{
		{
			name:         "ok/no_error",
			err:          nil,
			registryType: internal.RegistryTypeGeneric,
			want:         false,
		},
		{
			name:         "ok/http_400_bad_request",
			err:          &internal.HTTPError{StatusCode: http.StatusBadRequest, Message: "bad request"},
			registryType: internal.RegistryTypeGeneric,
			want:         true,
		},
		{
			name:         "ok/http_415_unsupported_media_type",
			err:          &internal.HTTPError{StatusCode: http.StatusUnsupportedMediaType, Message: "unsupported media type"},
			registryType: internal.RegistryTypeGeneric,
			want:         true,
		},
		{
			name:         "ok/http_501_not_implemented",
			err:          &internal.HTTPError{StatusCode: http.StatusNotImplemented, Message: "not implemented"},
			registryType: internal.RegistryTypeGeneric,
			want:         true,
		},
		{
			name:         "ok/http_502_bad_gateway",
			err:          &internal.HTTPError{StatusCode: http.StatusBadGateway, Message: "bad gateway"},
			registryType: internal.RegistryTypeGeneric,
			want:         true,
		},
		{
			name:         "ok/unsupported_manifest_type",
			err:          errors.New("unsupported manifest type"),
			registryType: internal.RegistryTypeGeneric,
			want:         true,
		},
		{
			name:         "ok/unsupported_media_type_message",
			err:          errors.New("unsupported media type: application/vnd.oci.artifact.manifest.v1+json"),
			registryType: internal.RegistryTypeGeneric,
			want:         true,
		},
		{
			name:         "ok/unknown_manifest_schema",
			err:          errors.New("unknown manifest schema"),
			registryType: internal.RegistryTypeGeneric,
			want:         true,
		},
		{
			name:         "ok/artifact_manifest_not_supported",
			err:          errors.New("artifact manifest not supported"),
			registryType: internal.RegistryTypeGeneric,
			want:         true,
		},
		{
			name:         "ok/ecr_manifest_blob_unknown",
			err:          errors.New("manifest blob unknown"),
			registryType: internal.RegistryTypeECR,
			want:         true,
		},
		{
			name:         "ok/ecr_unsupported_manifest_media_type",
			err:          errors.New("unsupported manifest media type"),
			registryType: internal.RegistryTypeECR,
			want:         true,
		},
		{
			name:         "ok/docker_hub_invalid_json",
			err:          errors.New("invalid json"),
			registryType: internal.RegistryTypeDockerHub,
			want:         true,
		},
		{
			name:         "ok/docker_hub_unknown_blob",
			err:          errors.New("unknown blob"),
			registryType: internal.RegistryTypeDockerHub,
			want:         true,
		},
		{
			name:         "error/http_404_not_found",
			err:          &internal.HTTPError{StatusCode: http.StatusNotFound, Message: "not found"},
			registryType: internal.RegistryTypeGeneric,
			want:         false,
		},
		{
			name:         "error/http_401_unauthorized",
			err:          &internal.HTTPError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"},
			registryType: internal.RegistryTypeGeneric,
			want:         false,
		},
		{
			name:         "error/generic_network_error",
			err:          errors.New("network timeout"),
			registryType: internal.RegistryTypeGeneric,
			want:         false,
		},
		{
			name:         "error/unrelated_error",
			err:          errors.New("some other error"),
			registryType: internal.RegistryTypeGeneric,
			want:         false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range var
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := internal.ShouldFallbackToImageManifest(tc.err, tc.registryType)
			assert.Equal(t, tc.want, got, "fallback decision failed for error=%v registry=%s", tc.err, tc.registryType.String())
		})
	}
}

func TestGetRegistrySpecificOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		registryType   internal.RegistryType
		wantPrefer     bool
		wantFallback   bool
	}{
		{
			name:           "ok/ghcr_excellent_support",
			registryType:   internal.RegistryTypeGHCR,
			wantPrefer:     true,
			wantFallback:   true,
		},
		{
			name:           "ok/gcr_good_support",
			registryType:   internal.RegistryTypeGCR,
			wantPrefer:     true,
			wantFallback:   true,
		},
		{
			name:           "ok/acr_supports_artifacts",
			registryType:   internal.RegistryTypeACR,
			wantPrefer:     true,
			wantFallback:   true,
		},
		{
			name:           "ok/quay_supports_artifacts",
			registryType:   internal.RegistryTypeQuay,
			wantPrefer:     true,
			wantFallback:   true,
		},
		{
			name:           "ok/ecr_variable_support",
			registryType:   internal.RegistryTypeECR,
			wantPrefer:     true,
			wantFallback:   true,
		},
		{
			name:           "ok/docker_hub_limited_support",
			registryType:   internal.RegistryTypeDockerHub,
			wantPrefer:     false,
			wantFallback:   true,
		},
		{
			name:           "ok/generic_try_both",
			registryType:   internal.RegistryTypeGeneric,
			wantPrefer:     true,
			wantFallback:   true,
		},
		{
			name:           "ok/unknown_try_both",
			registryType:   internal.RegistryTypeUnknown,
			wantPrefer:     true,
			wantFallback:   true,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range var
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			prefer, fallback := internal.GetRegistrySpecificOptions(tc.registryType)
			assert.Equal(t, tc.wantPrefer, prefer, "prefer artifact setting failed for registry=%s", tc.registryType.String())
			assert.Equal(t, tc.wantFallback, fallback, "fallback setting failed for registry=%s", tc.registryType.String())
		})
	}
}

func TestECRRegistryHelpers(t *testing.T) {
	t.Parallel()

	t.Run("ok/is_ecr_registry", func(t *testing.T) {
		t.Parallel()
		
		tests := []struct {
			registry string
			want     bool
		}{
			{"123456789012.dkr.ecr.us-east-1.amazonaws.com", true},
			{"123456789012.dkr.ecr.eu-west-1.amazonaws.com", true},
			{"ghcr.io", false},
			{"docker.io", false},
			{"localhost:5000", false},
		}

		for _, tc := range tests {
			got := internal.IsECRRegistry(tc.registry)
			assert.Equal(t, tc.want, got, "ECR detection failed for registry=%s", tc.registry)
		}
	})

	t.Run("ok/get_ecr_region", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			registry string
			want     string
		}{
			{"123456789012.dkr.ecr.us-east-1.amazonaws.com", "us-east-1"},
			{"123456789012.dkr.ecr.eu-west-1.amazonaws.com", "eu-west-1"},
			{"123456789012.dkr.ecr.ap-southeast-2.amazonaws.com", "ap-southeast-2"},
			{"invalid-registry", "us-east-1"}, // fallback
		}

		for _, tc := range tests {
			got := internal.GetECRRegion(tc.registry)
			assert.Equal(t, tc.want, got, "ECR region extraction failed for registry=%s", tc.registry)
		}
	})

	t.Run("ok/ecr_auth_helper", func(t *testing.T) {
		t.Parallel()

		helper := internal.NewECRAuthHelper("123456789012.dkr.ecr.us-west-2.amazonaws.com")
		require.NotNil(t, helper, "ECR auth helper should be created")
		
		assert.Equal(t, "123456789012", helper.AccountID, "account ID should be extracted")
		assert.Equal(t, "us-west-2", helper.Region, "region should be extracted")
		assert.True(t, helper.ShouldUseECRCredentialHelper(), "should use ECR credential helper")
		
		expectedEndpoint := "https://ecr.us-west-2.amazonaws.com"
		assert.Equal(t, expectedEndpoint, helper.GetECREndpoint(), "ECR endpoint should be correct")
	})
}

func TestGHCRRegistryHelpers(t *testing.T) {
	t.Parallel()

	t.Run("ok/is_ghcr_registry", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			registry string
			want     bool
		}{
			{"ghcr.io", true},
			{"GHCR.IO", true}, // case insensitive
			{"docker.io", false},
			{"gcr.io", false},
		}

		for _, tc := range tests {
			got := internal.IsGHCRRegistry(tc.registry)
			assert.Equal(t, tc.want, got, "GHCR detection failed for registry=%s", tc.registry)
		}
	})

	t.Run("ok/ghcr_optimize", func(t *testing.T) {
		t.Parallel()

		opts := internal.OptimizeForGHCR()
		assert.True(t, opts.PreferArtifacts, "GHCR should prefer artifacts")
		assert.True(t, opts.RequiresAuthentication, "GHCR typically requires auth")
		assert.Greater(t, opts.OptimalTimeout.Minutes(), float64(2), "GHCR timeout should be reasonable")
	})

	t.Run("ok/ghcr_auth_helper", func(t *testing.T) {
		t.Parallel()

		helper := internal.NewGHCRAuthHelper("ghp_token123", "testuser")
		require.NotNil(t, helper, "GHCR auth helper should be created")
		
		assert.Equal(t, "ghp_token123", helper.Token, "token should be set")
		assert.Equal(t, "testuser", helper.Username, "username should be set")
		
		// Test namespace extraction
		namespace := helper.ExtractNamespace("ghcr.io/testuser/repo:latest")
		assert.Equal(t, "testuser", namespace, "namespace should be extracted correctly")
		
		// Test public namespace detection
		assert.True(t, helper.IsPublicNamespace("library"), "library should be public")
		assert.True(t, helper.IsPublicNamespace("microsoft"), "microsoft should be public")
		assert.False(t, helper.IsPublicNamespace("privateuser"), "private user should not be public")
	})
}

func TestRegistryTypeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		registryType internal.RegistryType
		want         string
	}{
		{
			name:         "ok/docker_hub",
			registryType: internal.RegistryTypeDockerHub,
			want:         "docker.io",
		},
		{
			name:         "ok/ecr",
			registryType: internal.RegistryTypeECR,
			want:         "ecr",
		},
		{
			name:         "ok/ghcr",
			registryType: internal.RegistryTypeGHCR,
			want:         "ghcr.io",
		},
		{
			name:         "ok/gcr",
			registryType: internal.RegistryTypeGCR,
			want:         "gcr.io",
		},
		{
			name:         "ok/acr",
			registryType: internal.RegistryTypeACR,
			want:         "azurecr.io",
		},
		{
			name:         "ok/quay",
			registryType: internal.RegistryTypeQuay,
			want:         "quay.io",
		},
		{
			name:         "ok/generic",
			registryType: internal.RegistryTypeGeneric,
			want:         "generic",
		},
		{
			name:         "ok/unknown",
			registryType: internal.RegistryTypeUnknown,
			want:         "unknown",
		},
	}

	for _, tc := range tests {
		tc := tc // capture range var
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.registryType.String()
			assert.Equal(t, tc.want, got, "string representation failed for registry type")
		})
	}
}
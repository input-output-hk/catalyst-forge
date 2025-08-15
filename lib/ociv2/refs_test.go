package ociv2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDigestRef(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		ref      string
		expected bool
	}{
		{"sha256_digest", "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789", true},
		{"sha512_digest", "example.com/repo@sha512:def456", true},
		{"tag_reference", "example.com/repo:latest", false},
		{"no_tag_no_digest", "example.com/repo", false},
		{"empty_string", "", false},
		{"partial_sha", "example.com/repo@sha", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDigestRef(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureDigest(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		ref      string
		desc     Descriptor
		expected string
	}{
		{
			name: "ref_already_has_digest",
			ref:  "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			desc: Descriptor{Digest: "sha256:def4567890abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
			expected: "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		},
		{
			name: "ref_with_tag_add_digest",
			ref:  "example.com/repo:latest",
			desc: Descriptor{Digest: "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"},
			expected: "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		},
		{
			name: "ref_no_tag_add_digest",
			ref:  "example.com/repo",
			desc: Descriptor{Digest: "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"},
			expected: "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		},
		{
			name: "ref_with_port_add_digest",
			ref:  "localhost:5000/repo:v1.0",
			desc: Descriptor{Digest: "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"},
			expected: "localhost:5000/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		},
		{
			name: "no_digest_available",
			ref:  "example.com/repo:latest",
			desc: Descriptor{},
			expected: "example.com/repo:latest",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ensureDigest(tt.ref, tt.desc)
			assert.Equal(t, tt.expected, result.Ref)
		})
	}
}

func TestNormalizeRef(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "oci_scheme_removal",
			ref:      "oci://example.com/repo:latest",
			expected: "example.com/repo:latest",
		},
		{
			name:     "no_tag_add_latest",
			ref:      "example.com/repo",
			expected: "example.com/repo:latest",
		},
		{
			name:     "already_has_tag",
			ref:      "example.com/repo:v1.0",
			expected: "example.com/repo:v1.0",
		},
		{
			name:     "has_digest",
			ref:      "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			expected: "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		},
		{
			name:     "localhost_with_port",
			ref:      "localhost:5000/repo",
			expected: "localhost:5000/repo:latest",
		},
		{
			name:     "registry_with_port_and_tag",
			ref:      "registry.example.com:443/repo:v1.0",
			expected: "registry.example.com:443/repo:v1.0",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeRef(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToCanonical(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		ref      string
		digest   string
		expected string
	}{
		{
			name:     "add_digest_to_tagged_ref",
			ref:      "example.com/repo:latest",
			digest:   "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			expected: "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		},
		{
			name:     "replace_existing_digest",
			ref:      "example.com/repo@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			digest:   "sha256:fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
			expected: "example.com/repo@sha256:fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
		},
		{
			name:     "no_digest_provided",
			ref:      "example.com/repo:latest",
			digest:   "",
			expected: "example.com/repo:latest",
		},
		{
			name:     "registry_with_port",
			ref:      "localhost:5000/repo:v1.0",
			digest:   "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			expected: "localhost:5000/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toCanonical(tt.ref, tt.digest)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractRegistry(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		ref      string
		expected string
		wantErr  bool
	}{
		{
			name:     "docker_hub_implicit",
			ref:      "nginx:latest",
			expected: "index.docker.io",
			wantErr:  false,
		},
		{
			name:     "docker_hub_explicit",
			ref:      "docker.io/library/nginx:latest",
			expected: "index.docker.io",
			wantErr:  false,
		},
		{
			name:     "custom_registry",
			ref:      "registry.example.com/repo:latest",
			expected: "registry.example.com",
			wantErr:  false,
		},
		{
			name:     "registry_with_port",
			ref:      "localhost:5000/repo:latest",
			expected: "localhost:5000",
			wantErr:  false,
		},
		{
			name:     "gcr_registry",
			ref:      "gcr.io/project/image:tag",
			expected: "gcr.io",
			wantErr:  false,
		},
		{
			name:     "digest_reference",
			ref:      "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			expected: "example.com",
			wantErr:  false,
		},
		{
			name:     "invalid_reference",
			ref:      "INVALID!!reference@@format",
			expected: "",
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractRegistry(tt.ref)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidateRef(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name    string
		ref     string
		wantErr bool
		errType error
	}{
		{
			name:    "valid_tagged_ref",
			ref:     "example.com/repo:latest",
			wantErr: false,
		},
		{
			name:    "valid_digest_ref",
			ref:     "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			wantErr: false,
		},
		{
			name:    "valid_oci_scheme",
			ref:     "oci://example.com/repo:latest",
			wantErr: false,
		},
		{
			name:    "insecure_http",
			ref:     "http://example.com/repo:latest",
			wantErr: true,
			errType: ErrInsecureRef,
		},
		{
			name:    "invalid_format",
			ref:     "INVALID!!reference@@format",
			wantErr: true,
			errType: ErrInvalidRef,
		},
		{
			name:    "empty_reference",
			ref:     "",
			wantErr: true,
			errType: ErrInvalidRef,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRef(tt.ref)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsLocalRegistry(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		ref      string
		expected bool
	}{
		{
			name:     "localhost",
			ref:      "localhost:5000/repo:latest",
			expected: true,
		},
		{
			name:     "localhost_no_port",
			ref:      "localhost/repo:latest",
			expected: true,
		},
		{
			name:     "loopback_ipv4",
			ref:      "127.0.0.1:5000/repo:latest",
			expected: true,
		},
		{
			name:     "loopback_ipv6",
			ref:      "::1:5000/repo:latest",
			expected: true,
		},
		{
			name:     "docker_internal",
			ref:      "host.docker.internal:5000/repo:latest",
			expected: true,
		},
		{
			name:     "private_ip_10",
			ref:      "10.0.0.1:5000/repo:latest",
			expected: true,
		},
		{
			name:     "private_ip_172",
			ref:      "172.16.0.1:5000/repo:latest",
			expected: true,
		},
		{
			name:     "private_ip_192",
			ref:      "192.168.1.1:5000/repo:latest",
			expected: true,
		},
		{
			name:     "public_registry",
			ref:      "docker.io/library/nginx:latest",
			expected: false,
		},
		{
			name:     "custom_registry",
			ref:      "registry.example.com/repo:latest",
			expected: false,
		},
		{
			name:     "gcr",
			ref:      "gcr.io/project/image:tag",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalRegistry(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitRefParts(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name         string
		ref          string
		wantRegistry string
		wantRepo     string
		wantTag      string
		wantErr      bool
	}{
		{
			name:         "tagged_reference",
			ref:          "example.com/repo:latest",
			wantRegistry: "example.com",
			wantRepo:     "repo",
			wantTag:      "latest",
			wantErr:      false,
		},
		{
			name:         "digest_reference",
			ref:          "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			wantRegistry: "example.com",
			wantRepo:     "repo",
			wantTag:      "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			wantErr:      false,
		},
		{
			name:         "docker_hub_library",
			ref:          "nginx:latest",
			wantRegistry: "index.docker.io",
			wantRepo:     "library/nginx",
			wantTag:      "latest",
			wantErr:      false,
		},
		{
			name:         "nested_repository",
			ref:          "gcr.io/project/team/service:v1.0",
			wantRegistry: "gcr.io",
			wantRepo:     "project/team/service",
			wantTag:      "v1.0",
			wantErr:      false,
		},
		{
			name:         "registry_with_port",
			ref:          "localhost:5000/repo:latest",
			wantRegistry: "localhost:5000",
			wantRepo:     "repo",
			wantTag:      "latest",
			wantErr:      false,
		},
		{
			name:    "invalid_reference",
			ref:     "INVALID!!reference@@format",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, repo, tag, err := splitRefParts(tt.ref)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantRegistry, registry)
				assert.Equal(t, tt.wantRepo, repo)
				assert.Equal(t, tt.wantTag, tag)
			}
		})
	}
}

func TestMakeOCIURL(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "normal_reference",
			ref:      "example.com/repo:latest",
			expected: "oci://example.com/repo:latest",
		},
		{
			name:     "already_oci_scheme",
			ref:      "oci://example.com/repo:latest",
			expected: "oci://example.com/repo:latest",
		},
		{
			name:     "digest_reference",
			ref:      "example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			expected: "oci://example.com/repo@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := makeOCIURL(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsHTTPSRegistry(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name      string
		ref       string
		plainHTTP bool
		expected  bool
	}{
		{
			name:      "plain_http_enabled",
			ref:       "example.com/repo:latest",
			plainHTTP: true,
			expected:  false,
		},
		{
			name:      "local_registry_http",
			ref:       "localhost:5000/repo:latest",
			plainHTTP: false,
			expected:  false,
		},
		{
			name:      "remote_registry_https",
			ref:       "docker.io/library/nginx:latest",
			plainHTTP: false,
			expected:  true,
		},
		{
			name:      "private_ip_http",
			ref:       "192.168.1.100:5000/repo:latest",
			plainHTTP: false,
			expected:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHTTPSRegistry(tt.ref, tt.plainHTTP)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseRegistryURL(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name       string
		registry   string
		useHTTPS   bool
		wantScheme string
		wantHost   string
		wantErr    bool
	}{
		{
			name:       "https_registry",
			registry:   "example.com",
			useHTTPS:   true,
			wantScheme: "https",
			wantHost:   "example.com",
			wantErr:    false,
		},
		{
			name:       "http_registry",
			registry:   "localhost:5000",
			useHTTPS:   false,
			wantScheme: "http",
			wantHost:   "localhost:5000",
			wantErr:    false,
		},
		{
			name:       "registry_with_scheme",
			registry:   "https://registry.example.com",
			useHTTPS:   false, // Should be ignored since scheme is already present
			wantScheme: "https",
			wantHost:   "registry.example.com",
			wantErr:    false,
		},
		{
			name:     "invalid_url", 
			registry: "invalid url with spaces",
			useHTTPS: true,
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRegistryURL(tt.registry, tt.useHTTPS)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantScheme, result.Scheme)
				assert.Equal(t, tt.wantHost, result.Host)
			}
		})
	}
}

func TestParseRef(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name    string
		ref     string
		wantErr bool
	}{
		{
			name:    "valid_tag_ref",
			ref:     "example.com/repo:latest",
			wantErr: false,
		},
		{
			name:    "valid_digest_ref",
			ref:     "example.com/repo@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			wantErr: false,
		},
		{
			name:    "docker_hub_short",
			ref:     "nginx:latest",
			wantErr: false,
		},
		{
			name:    "oci_scheme",
			ref:     "oci://example.com/repo:latest",
			wantErr: false,
		},
		{
			name:    "invalid_digest",
			ref:     "example.com/repo@invalid",
			wantErr: true,
		},
		{
			name:    "empty_ref",
			ref:     "",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseRef(tt.ref)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
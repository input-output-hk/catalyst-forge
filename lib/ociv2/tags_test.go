package ociv2

import (
	"context"
	"fmt"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSemVer(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name    string
		tag     string
		want    *SemVer
		wantNil bool
	}{
		{
			name: "basic_version",
			tag:  "1.2.3",
			want: &SemVer{
				Major:    1,
				Minor:    2,
				Patch:    3,
				Original: "1.2.3",
			},
		},
		{
			name: "version_with_v_prefix",
			tag:  "v1.2.3",
			want: &SemVer{
				Major:    1,
				Minor:    2,
				Patch:    3,
				Original: "1.2.3",
			},
		},
		{
			name: "version_with_prerelease",
			tag:  "1.0.0-alpha",
			want: &SemVer{
				Major:      1,
				Minor:      0,
				Patch:      0,
				Prerelease: "alpha",
				Original:   "1.0.0-alpha",
			},
		},
		{
			name: "version_with_prerelease_and_number",
			tag:  "2.1.0-beta.1",
			want: &SemVer{
				Major:      2,
				Minor:      1,
				Patch:      0,
				Prerelease: "beta.1",
				Original:   "2.1.0-beta.1",
			},
		},
		{
			name: "version_with_build_metadata",
			tag:  "1.0.0+20130313144700",
			want: &SemVer{
				Major:    1,
				Minor:    0,
				Patch:    0,
				Build:    "20130313144700",
				Original: "1.0.0+20130313144700",
			},
		},
		{
			name: "version_with_prerelease_and_build",
			tag:  "1.0.0-rc.1+build.123",
			want: &SemVer{
				Major:      1,
				Minor:      0,
				Patch:      0,
				Prerelease: "rc.1",
				Build:      "build.123",
				Original:   "1.0.0-rc.1+build.123",
			},
		},
		{
			name:    "invalid_not_semver",
			tag:     "latest",
			wantNil: true,
		},
		{
			name:    "invalid_missing_minor",
			tag:     "1.2",
			wantNil: true,
		},
		{
			name:    "invalid_non_numeric",
			tag:     "a.b.c",
			wantNil: true,
		},
		{
			name:    "empty_string",
			tag:     "",
			wantNil: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSemVer(tt.tag)
			
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.want.Major, got.Major)
				assert.Equal(t, tt.want.Minor, got.Minor)
				assert.Equal(t, tt.want.Patch, got.Patch)
				assert.Equal(t, tt.want.Prerelease, got.Prerelease)
				assert.Equal(t, tt.want.Build, got.Build)
				assert.Equal(t, tt.want.Original, got.Original)
			}
		})
	}
}

func TestCompareSemVer(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name     string
		a        string
		b        string
		expected int // 1 if a > b, -1 if a < b, 0 if equal
	}{
		// Basic comparisons
		{"equal", "1.0.0", "1.0.0", 0},
		{"major_greater", "2.0.0", "1.0.0", 1},
		{"major_less", "1.0.0", "2.0.0", -1},
		{"minor_greater", "1.2.0", "1.1.0", 1},
		{"minor_less", "1.1.0", "1.2.0", -1},
		{"patch_greater", "1.0.2", "1.0.1", 1},
		{"patch_less", "1.0.1", "1.0.2", -1},
		
		// Prerelease comparisons
		{"release_vs_prerelease", "1.0.0", "1.0.0-alpha", 1},
		{"prerelease_vs_release", "1.0.0-alpha", "1.0.0", -1},
		{"alpha_vs_beta", "1.0.0-alpha", "1.0.0-beta", -1},
		{"beta_vs_alpha", "1.0.0-beta", "1.0.0-alpha", 1},
		{"rc_vs_beta", "1.0.0-rc", "1.0.0-beta", 1},
		{"numeric_prerelease", "1.0.0-1", "1.0.0-2", -1},
		{"alpha.1_vs_alpha.2", "1.0.0-alpha.1", "1.0.0-alpha.2", -1},
		{"fewer_prerelease_parts", "1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"more_prerelease_parts", "1.0.0-alpha.1", "1.0.0-alpha", 1},
		
		// Build metadata (should be ignored in comparison)
		{"same_with_different_build", "1.0.0+build1", "1.0.0+build2", 0},
		{"with_and_without_build", "1.0.0+build", "1.0.0", 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := parseSemVer(tt.a)
			b := parseSemVer(tt.b)
			
			require.NotNil(t, a)
			require.NotNil(t, b)
			
			result := compareSemVer(*a, *b)
			
			switch tt.expected {
			case 1:
				assert.Equal(t, 1, result, "%s should be > %s", tt.a, tt.b)
			case -1:
				assert.Equal(t, -1, result, "%s should be < %s", tt.a, tt.b)
			case 0:
				assert.Equal(t, 0, result, "%s should be == %s", tt.a, tt.b)
			}
		})
	}
}

func TestLatestSemverTag(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name              string
		tags              []string
		includePrerelease bool
		want              string
		wantErr           bool
	}{
		{
			name: "basic_versions",
			tags: []string{"v1.0.0", "v1.1.0", "v1.2.0", "v2.0.0"},
			want: "2.0.0",
		},
		{
			name: "with_non_semver_tags",
			tags: []string{"latest", "v1.0.0", "dev", "v2.1.0", "stable"},
			want: "2.1.0",
		},
		{
			name:              "exclude_prerelease",
			tags:              []string{"v1.0.0", "v2.0.0-rc.1", "v1.5.0"},
			includePrerelease: false,
			want:              "1.5.0",
		},
		{
			name:              "include_prerelease",
			tags:              []string{"v1.0.0", "v2.0.0-rc.1", "v1.5.0"},
			includePrerelease: true,
			want:              "2.0.0-rc.1",
		},
		{
			name: "complex_versions",
			tags: []string{
				"v0.1.0",
				"v0.2.0-alpha",
				"v0.2.0-beta.1",
				"v0.2.0-beta.2",
				"v0.2.0-rc.1",
				"v0.2.0",
				"v1.0.0-alpha.1",
				"v1.0.0",
				"v1.1.0",
			},
			includePrerelease: false,
			want:              "1.1.0",
		},
		{
			name:    "no_tags",
			tags:    []string{},
			wantErr: true,
		},
		{
			name:    "no_semver_tags",
			tags:    []string{"latest", "dev", "stable"},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock client with custom ListTags
			client := &mockClientWithTags{
				tags: tt.tags,
			}
			
			latest, err := client.LatestSemverTag(context.Background(), "test/repo", tt.includePrerelease)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, latest)
			}
		})
	}
}

// mockClientWithTags is a test helper that implements tag operations
type mockClientWithTags struct {
	tags []string
}

func (m *mockClientWithTags) ListTags(ctx context.Context, repo string) ([]string, error) {
	return m.tags, nil
}

func (m *mockClientWithTags) LatestSemverTag(ctx context.Context, repo string, includePrerelease bool) (string, error) {
	tags, err := m.ListTags(ctx, repo)
	if err != nil {
		return "", err
	}
	
	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found in repository")
	}
	
	var versions []SemVer
	for _, tag := range tags {
		if v := parseSemVer(tag); v != nil {
			if !includePrerelease && v.Prerelease != "" {
				continue
			}
			versions = append(versions, *v)
		}
	}
	
	if len(versions) == 0 {
		return "", fmt.Errorf("no valid semantic version tags found")
	}
	
	sort.Slice(versions, func(i, j int) bool {
		return compareSemVer(versions[i], versions[j]) > 0
	})
	
	return versions[0].Original, nil
}

func TestIntegrationListTags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	t.Parallel()
	
	// Set up in-memory registry
	registryServer := httptest.NewServer(registry.New())
	defer registryServer.Close()
	
	registryHost := strings.TrimPrefix(registryServer.URL, "http://")
	repo := fmt.Sprintf("%s/test/tags", registryHost)
	
	// Create client
	client, err := New(ClientOptions{
		PlainHTTP: true,
	})
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// Push some tagged images
	testTags := []string{"v1.0.0", "v1.1.0", "v2.0.0-beta", "latest", "dev"}
	
	for _, tag := range testTags {
		ref := fmt.Sprintf("%s:%s", repo, tag)
		
		// Create and push a random image
		img, err := random.Image(1024, 1)
		require.NoError(t, err)
		
		nameRef, err := name.ParseReference(ref)
		require.NoError(t, err)
		
		err = remote.Write(nameRef, img, 
			remote.WithPlatform(v1.Platform{
				OS:           "linux",
				Architecture: "amd64",
			}))
		require.NoError(t, err)
	}
	
	t.Run("ListAllTags", func(t *testing.T) {
		tags, err := client.ListTags(ctx, repo)
		require.NoError(t, err)
		
		// Should have all tags
		assert.Len(t, tags, len(testTags))
		
		// Check all tags are present
		for _, expectedTag := range testTags {
			assert.Contains(t, tags, expectedTag)
		}
	})
	
	t.Run("LatestSemverTag_ExcludePrerelease", func(t *testing.T) {
		latest, err := client.LatestSemverTag(ctx, repo, false)
		require.NoError(t, err)
		assert.Equal(t, "1.1.0", latest)
	})
	
	t.Run("LatestSemverTag_IncludePrerelease", func(t *testing.T) {
		latest, err := client.LatestSemverTag(ctx, repo, true)
		require.NoError(t, err)
		assert.Equal(t, "2.0.0-beta", latest)
	})
}

func TestListTagsWithOptions(t *testing.T) {
	t.Parallel()
	
	allTags := []string{
		"v1.0.0",
		"v1.1.0",
		"v1.2.0",
		"v2.0.0-alpha",
		"v2.0.0-beta",
		"v2.0.0",
		"latest",
		"dev",
		"stable",
		"nightly-20240101",
		"nightly-20240102",
	}
	
	client := &testClient{
		mockListTags: func(ctx context.Context, repo string) ([]string, error) {
			return allTags, nil
		},
	}
	
	ctx := context.Background()
	
	tests := []struct {
		name string
		opts TagListOptions
		want []string
	}{
		{
			name: "no_filter",
			opts: TagListOptions{},
			want: allTags,
		},
		{
			name: "semver_only",
			opts: TagListOptions{
				SemverOnly: true,
			},
			want: []string{"v1.0.0", "v1.1.0", "v1.2.0", "v2.0.0-alpha", "v2.0.0-beta", "v2.0.0"},
		},
		{
			name: "pattern_filter",
			opts: TagListOptions{
				Pattern: "^v1\\.",
			},
			want: []string{"v1.0.0", "v1.1.0", "v1.2.0"},
		},
		{
			name: "pattern_nightly",
			opts: TagListOptions{
				Pattern: "^nightly-",
			},
			want: []string{"nightly-20240101", "nightly-20240102"},
		},
		{
			name: "limit",
			opts: TagListOptions{
				Limit: 3,
			},
			want: []string{"v1.0.0", "v1.1.0", "v1.2.0"},
		},
		{
			name: "semver_with_limit",
			opts: TagListOptions{
				SemverOnly: true,
				Limit:      2,
			},
			want: []string{"v1.0.0", "v1.1.0"},
		},
		{
			name: "pattern_with_limit",
			opts: TagListOptions{
				Pattern: "^v2",
				Limit:   2,
			},
			want: []string{"v2.0.0-alpha", "v2.0.0-beta"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.ListTagsWithOptions(ctx, "test/repo", tt.opts)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// testClient is a test implementation of the client
type testClient struct {
	client
	mockListTags func(ctx context.Context, repo string) ([]string, error)
}

func (tc *testClient) ListTags(ctx context.Context, repo string) ([]string, error) {
	if tc.mockListTags != nil {
		return tc.mockListTags(ctx, repo)
	}
	return tc.client.ListTags(ctx, repo)
}

func (tc *testClient) ListTagsWithOptions(ctx context.Context, repo string, opts TagListOptions) ([]string, error) {
	tags, err := tc.ListTags(ctx, repo)
	if err != nil {
		return nil, err
	}
	
	filtered := []string{}
	for _, tag := range tags {
		if opts.SemverOnly && parseSemVer(tag) == nil {
			continue
		}
		
		if opts.Pattern != "" {
			matched, _ := regexp.MatchString(opts.Pattern, tag)
			if !matched {
				continue
			}
		}
		
		filtered = append(filtered, tag)
		
		if opts.Limit > 0 && len(filtered) >= opts.Limit {
			break
		}
	}
	
	return filtered, nil
}
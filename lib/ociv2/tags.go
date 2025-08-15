package ociv2

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	ggcrremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/observability"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// SemVer represents a parsed semantic version
type SemVer struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
	Original   string
}

// ListTags lists all tags for a repository
func (c *client) ListTags(ctx context.Context, repo string) ([]string, error) {
	operation := "list_tags"

	// Validate repository format
	if repo == "" {
		return nil, observability.NewValidationError(operation, repo, fmt.Errorf("repository cannot be empty"))
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	// Track operation
	tracker := &OperationTracker{
		StartTime: time.Now(),
		Operation: operation,
		Fields:    map[string]interface{}{"repo": repo},
	}
	defer func() {
		// Metrics recording handled where durations are known
	}()

	// Normalize repository (remove oci:// prefix if present)
	repo = NormalizeRef(repo)

	// Extract registry
	registryHost, err := extractRegistry(repo)
	if err != nil {
		return nil, observability.NewValidationError(operation, repo, fmt.Errorf("failed to extract registry: %w", err))
	}

	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.list_tags", "repo", repo)
	}

	// Try ORAS first
	tags, err := c.listTagsORAS(ctx, repo, registryHost)
	if err == nil {
		tracker.Fields["backend"] = "oras"
		tracker.Fields["tags"] = len(tags)
		c.recordMetrics(operation, registryHost, time.Since(tracker.StartTime), nil)
		return tags, nil
	}

	// Fallback to ggcr
	tags, err2 := c.listTagsGGCR(ctx, repo, registryHost)
	if err2 == nil {
		tracker.Fields["backend"] = "ggcr"
		tracker.Fields["tags"] = len(tags)
		tracker.Fields["fallback"] = true
		c.recordMetrics(operation, registryHost, time.Since(tracker.StartTime), nil)
		return tags, nil
	}

	// Both failed
	finalErr := c.wrapError(err, operation, repo, registryHost)
	c.recordMetrics(operation, registryHost, time.Since(tracker.StartTime), finalErr)
	return nil, finalErr
}

// listTagsORAS lists tags using ORAS
func (c *client) listTagsORAS(ctx context.Context, repo, registryHost string) ([]string, error) {
	// Create repository client
	repoClient, err := remote.NewRepository(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Configure plain HTTP if needed
	if c.opts.PlainHTTP {
		repoClient.PlainHTTP = true
	}

	// Set up auth
	authFunc := c.getORASAuth()
	if authFunc != nil {
		repoClient.Client = &auth.Client{
			Credential: authFunc,
		}
	}

	// List tags
	tags := []string{}
	err = repoClient.Tags(ctx, "", func(tagList []string) error {
		tags = append(tags, tagList...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, nil
}

// listTagsGGCR lists tags using go-containerregistry
func (c *client) listTagsGGCR(ctx context.Context, repo, registryHost string) ([]string, error) {
	// Parse repository
	repoRef, err := name.NewRepository(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository: %w", err)
	}

	// Get auth
	authFunc := c.getGGCRAuthFor(registryHost)

	// Set up remote options
	remoteOpts := []ggcrremote.Option{
		ggcrremote.WithContext(ctx),
		ggcrremote.WithUserAgent(c.opts.UserAgent),
	}
	if authFunc != nil {
		auth, err := authFunc()
		if err == nil && auth != nil {
			remoteOpts = append(remoteOpts, ggcrremote.WithAuth(auth))
		}
	}
	if c.opts.PlainHTTP {
		remoteOpts = append(remoteOpts, ggcrremote.WithTransport(c.transport))
	}

	// List tags
	tags, err := ggcrremote.List(repoRef, remoteOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, nil
}

// LatestSemverTag returns the latest semantic version tag
func (c *client) LatestSemverTag(ctx context.Context, repo string, includePrerelease bool) (string, error) {

	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.latest_semver_tag", "repo", repo, "includePrerelease", includePrerelease)
	}

	// List all tags
	tags, err := c.ListTags(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("failed to list tags: %w", err)
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found in repository")
	}

	// Parse semver tags
	var versions []SemVer
	for _, tag := range tags {
		if v := parseSemVer(tag); v != nil {
			// Skip prereleases if not included
			if !includePrerelease && v.Prerelease != "" {
				continue
			}
			versions = append(versions, *v)
		}
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no valid semantic version tags found")
	}

	// Sort versions (latest first)
	sort.Slice(versions, func(i, j int) bool {
		return compareSemVer(versions[i], versions[j]) > 0
	})

	latest := versions[0].Original

	if c.opts.Logger != nil {
		c.opts.Logger("oci.latest_semver_tag.found", "repo", repo, "latest", latest, "total", len(versions))
	}
	return latest, nil
}

// parseSemVer parses a semantic version string
func parseSemVer(tag string) *SemVer {
	// Remove 'v' prefix if present
	tag = strings.TrimPrefix(tag, "v")
	tag = strings.TrimPrefix(tag, "V")

	// Regular expression for semantic versioning
	// Matches: MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z\-\.]+))?(?:\+([0-9A-Za-z\-\.]+))?$`)

	matches := re.FindStringSubmatch(tag)
	if matches == nil {
		return nil
	}

	v := &SemVer{
		Original: tag,
	}

	// Parse major, minor, patch
	if _, err := fmt.Sscanf(matches[1], "%d", &v.Major); err != nil {
		return nil
	}
	if _, err := fmt.Sscanf(matches[2], "%d", &v.Minor); err != nil {
		return nil
	}
	if _, err := fmt.Sscanf(matches[3], "%d", &v.Patch); err != nil {
		return nil
	}

	// Parse prerelease and build metadata
	if len(matches) > 4 {
		v.Prerelease = matches[4]
	}
	if len(matches) > 5 {
		v.Build = matches[5]
	}

	return v
}

// compareSemVer compares two semantic versions
// Returns: 1 if a > b, -1 if a < b, 0 if a == b
func compareSemVer(a, b SemVer) int {
	// Compare major
	if a.Major != b.Major {
		if a.Major > b.Major {
			return 1
		}
		return -1
	}

	// Compare minor
	if a.Minor != b.Minor {
		if a.Minor > b.Minor {
			return 1
		}
		return -1
	}

	// Compare patch
	if a.Patch != b.Patch {
		if a.Patch > b.Patch {
			return 1
		}
		return -1
	}

	// Compare prerelease
	// No prerelease > prerelease (1.0.0 > 1.0.0-alpha)
	if a.Prerelease == "" && b.Prerelease != "" {
		return 1
	}
	if a.Prerelease != "" && b.Prerelease == "" {
		return -1
	}

	// Compare prerelease identifiers
	if a.Prerelease != b.Prerelease {
		return comparePrereleaseVersions(a.Prerelease, b.Prerelease)
	}

	return 0
}

// comparePrereleaseVersions compares prerelease version strings
func comparePrereleaseVersions(a, b string) int {
	// Split by dots
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	// Compare each part
	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		aPart := aParts[i]
		bPart := bParts[i]

		// Try to parse as numbers
		var aNum, bNum int
		_, aErr := fmt.Sscanf(aPart, "%d", &aNum)
		_, bErr := fmt.Sscanf(bPart, "%d", &bNum)
		aIsNum := aErr == nil
		bIsNum := bErr == nil

		// Both numeric
		if aIsNum && bIsNum {
			if aNum != bNum {
				if aNum > bNum {
					return 1
				}
				return -1
			}
			continue
		}

		// Numeric < non-numeric
		if aIsNum && !bIsNum {
			return -1
		}
		if !aIsNum && bIsNum {
			return 1
		}

		// Both non-numeric, compare as strings
		if aPart != bPart {
			if aPart > bPart {
				return 1
			}
			return -1
		}
	}

	// Fewer parts < more parts
	if len(aParts) < len(bParts) {
		return -1
	}
	if len(aParts) > len(bParts) {
		return 1
	}

	return 0
}

// TagListOptions provides options for listing tags
type TagListOptions struct {
	// Filter tags by pattern (e.g., "v1.*")
	Pattern string

	// Maximum number of tags to return (0 = all)
	Limit int

	// Include only semver-compliant tags
	SemverOnly bool
}

// ListTagsWithOptions lists tags with filtering options
func (c *client) ListTagsWithOptions(ctx context.Context, repo string, opts TagListOptions) ([]string, error) {
	// Get all tags
	tags, err := c.ListTags(ctx, repo)
	if err != nil {
		return nil, err
	}

	// Apply filters
	filtered := []string{}
	for _, tag := range tags {
		// Check semver filter
		if opts.SemverOnly && parseSemVer(tag) == nil {
			continue
		}

		// Check pattern filter
		if opts.Pattern != "" {
			matched, _ := regexp.MatchString(opts.Pattern, tag)
			if !matched {
				continue
			}
		}

		filtered = append(filtered, tag)

		// Check limit
		if opts.Limit > 0 && len(filtered) >= opts.Limit {
			break
		}
	}

	return filtered, nil
}

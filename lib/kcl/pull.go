package kcl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/kcl/cache"
	"github.com/input-output-hk/catalyst-forge/lib/kcl/internal"
	"github.com/input-output-hk/catalyst-forge/lib/kcl/logging"
	"github.com/input-output-hk/catalyst-forge/lib/kcl/metrics"
)

// Pull pulls a KCL module and caches it locally.
func Pull(ctx context.Context, ociCli OCI, ref ModuleRef, opts PullOptions) (*PullResult, error) {
	start := time.Now()
	log := logging.GetLogger().WithContext(ctx).With(
		"repo", ref.Repo,
		"tag", ref.Tag,
		"digest", ref.Dig,
		"profile", opts.Profile,
	)

	log.Info("Starting module pull")
	metrics.GetMetrics().ModulePulls.Inc()

	// Measure pull duration
	defer func() {
		metrics.GetMetrics().PullDuration.Observe(time.Since(start).Seconds())
	}()

	// Get cache manager
	cm, err := cache.GetManager()
	if err != nil {
		log.Error("Failed to get cache manager", "error", err)
		return nil, fmt.Errorf("failed to get cache manager: %w", err)
	}

	// Verify the module first (unless we have a digest and trust it)
	var digest string
	var meta []byte

	if ref.Dig != "" && opts.SkipVerify {
		// Fast path: we have digest and skip verification
		digest = normalizeDigest(ref.Dig)
	} else {
		// Full verification path
		verifyOpts := VerifyOptions{
			Profile:          opts.Profile,
			RequireSignature: opts.RequireSignature,
		}
		digest, meta, err = Verify(ctx, ociCli, ref, verifyOpts)
		if err != nil {
			return nil, fmt.Errorf("verification failed: %w", err)
		}
	}

	// Check if module is already cached
	moduleDir := cm.ModulePath(stripDigestPrefix(digest))
	metaPath := filepath.Join(moduleDir, ".meta.json")

	if internal.DirExists(moduleDir) && internal.FileExists(metaPath) {
		// Module exists in cache, touch stamp and return
		if err := cm.TouchStamp(moduleDir); err != nil {
			// Non-fatal error
			fmt.Fprintf(os.Stderr, "warning: failed to update cache timestamp: %v\n", err)
		}

		// Read cached metadata if we don't have it
		if meta == nil {
			meta, err = os.ReadFile(metaPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read cached metadata: %w", err)
			}
		}

		// Parse metadata to ModuleMeta
		var moduleMeta ModuleMeta
		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &moduleMeta); err != nil {
				return nil, fmt.Errorf("failed to parse module metadata: %w", err)
			}
		}

		return &PullResult{
			Path:     moduleDir,
			Digest:   digest,
			Meta:     &moduleMeta,
			MetaJSON: meta,
		}, nil
	}

	// Need to pull the module - use singleflight to dedupe
	result, err := cm.SingleflightModule(digest, func() (interface{}, error) {
		return pullModule(ctx, ociCli, cm, ref, digest, meta, opts)
	})

	if err != nil {
		return nil, err
	}

	// Unpack result
	res := result.(*pullResult)

	// Parse metadata to ModuleMeta
	var moduleMeta ModuleMeta
	if len(res.meta) > 0 {
		if err := json.Unmarshal(res.meta, &moduleMeta); err != nil {
			return nil, fmt.Errorf("failed to parse module metadata: %w", err)
		}
	}

	return &PullResult{
		Path:     res.path,
		Digest:   res.digest,
		Meta:     &moduleMeta,
		MetaJSON: res.meta,
	}, nil
}

// pullResult holds the result of a pull operation.
type pullResult struct {
	path   string
	digest string
	meta   []byte
}

// pullModule performs the actual module pull with locking.
func pullModule(ctx context.Context, ociCli OCI, cm *cache.Manager, ref ModuleRef, digest string, meta []byte, opts PullOptions) (*pullResult, error) {
	moduleDir := cm.ModulePath(stripDigestPrefix(digest))

	// Acquire cross-process lock
	err := cm.WithModuleLock(stripDigestPrefix(digest), func() error {
		// Double-check after acquiring lock
		metaPath := filepath.Join(moduleDir, ".meta.json")
		if internal.DirExists(moduleDir) && internal.FileExists(metaPath) {
			// Another process created it while we waited
			if meta == nil {
				var err error
				meta, err = os.ReadFile(metaPath)
				if err != nil {
					return fmt.Errorf("failed to read metadata: %w", err)
				}
			}
			return nil
		}

		// Pull the artifact
		refStr := buildReferenceWithDigest(ref.Repo, digest)
		pullResult, err := ociCli.PullArtifact(refStr)
		if err != nil {
			return fmt.Errorf("failed to pull artifact: %w", err)
		}

		// Find the tar layer based on profile
		tarData, err := extractTarLayer(pullResult, opts.Profile)
		if err != nil {
			return fmt.Errorf("failed to extract tar layer: %w", err)
		}

		// Extract to module directory
		if err := internal.ExtractSafe(bytes.NewReader(tarData), moduleDir); err != nil {
			// Clean up on failure
			_ = os.RemoveAll(moduleDir)
			return fmt.Errorf("failed to extract module: %w", err)
		}

		// Generate metadata if needed
		if meta == nil {
			meta, err = generateMetadataFromPull(pullResult, opts.Profile)
			if err != nil {
				// Clean up on failure
				_ = os.RemoveAll(moduleDir)
				return fmt.Errorf("failed to generate metadata: %w", err)
			}
		}

		// Write metadata
		if err := os.WriteFile(metaPath, meta, 0644); err != nil {
			// Clean up on failure
			_ = os.RemoveAll(moduleDir)
			return fmt.Errorf("failed to write metadata: %w", err)
		}

		// Update cache size and enforce limits
		if err := cm.EnforceLimits("modules"); err != nil {
			// Non-fatal error
			fmt.Fprintf(os.Stderr, "warning: failed to enforce cache limits: %v\n", err)
		}

		// Save to blob cache if enabled
		if cm.EnableBlobCache {
			blobPath := cm.BlobPath(stripDigestPrefix(digest))
			if err := os.WriteFile(blobPath, tarData, 0644); err != nil {
				// Non-fatal error
				fmt.Fprintf(os.Stderr, "warning: failed to save blob cache: %v\n", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pullResult{
		path:   moduleDir,
		digest: digest,
		meta:   meta,
	}, nil
}

// extractTarLayer extracts the tar layer from pull result based on profile.
func extractTarLayer(result *OCIPullResult, profile Profile) ([]byte, error) {
	// Determine expected media type based on profile
	var expectedMediaType string
	switch profile {
	case ProfileCompat:
		expectedMediaType = "application/vnd.oci.image.layer.v1.tar"
	case ProfileStrict:
		expectedMediaType = "application/vnd.projectcatalyst.kcl.module.tar.v1"
	default:
		return nil, fmt.Errorf("unsupported profile: %s", profile)
	}

	// Find the tar layer
	for _, layer := range result.Manifest.Layers {
		if layer.MediaType == expectedMediaType {
			// Get layer content by digest
			digest := strings.TrimPrefix(string(layer.Digest), "sha256:")
			if data, exists := result.Layers[digest]; exists {
				return data, nil
			}
			// Try with full digest
			if data, exists := result.Layers[string(layer.Digest)]; exists {
				return data, nil
			}
		}
	}

	return nil, fmt.Errorf("tar layer not found for profile %s", profile)
}

// generateMetadataFromPull generates metadata from pull result.
func generateMetadataFromPull(result *OCIPullResult, profile Profile) ([]byte, error) {
	switch profile {
	case ProfileCompat:
		// Generate from annotations
		meta := ModuleMeta{
			Annotations: result.Annotations,
		}

		// Extract standard fields
		if name, exists := result.Annotations["io.kcl.name"]; exists {
			meta.Name = name
		} else if name, exists := result.Annotations["org.opencontainers.image.title"]; exists {
			meta.Name = name
		}

		if version, exists := result.Annotations["io.kcl.version"]; exists {
			meta.Version = version
		} else if version, exists := result.Annotations["org.opencontainers.image.version"]; exists {
			meta.Version = version
		}

		if desc, exists := result.Annotations["org.opencontainers.image.description"]; exists {
			meta.Description = desc
		}

		if sum, exists := result.Annotations["io.kcl.sum"]; exists {
			meta.Sum = sum
		}

		return json.Marshal(meta)

	case ProfileStrict:
		// Extract from meta.json layer
		metaMediaType := "application/vnd.projectcatalyst.kcl.module.meta.v1+json"

		for _, layer := range result.Manifest.Layers {
			if layer.MediaType == metaMediaType {
				digest := strings.TrimPrefix(string(layer.Digest), "sha256:")
				if data, exists := result.Layers[digest]; exists {
					return data, nil
				}
				if data, exists := result.Layers[string(layer.Digest)]; exists {
					return data, nil
				}
			}
		}

		return nil, fmt.Errorf("meta.json layer not found in strict profile artifact")

	default:
		return nil, fmt.Errorf("unsupported profile: %s", profile)
	}
}

// buildReferenceWithDigest builds a reference string with digest.
func buildReferenceWithDigest(repo, digest string) string {
	repo = strings.TrimPrefix(repo, "oci://")
	return fmt.Sprintf("%s@%s", repo, normalizeDigest(digest))
}

// stripDigestPrefix removes the sha256: prefix from a digest.
func stripDigestPrefix(digest string) string {
	return strings.TrimPrefix(digest, "sha256:")
}

// parseSize parses a size string (simplified).
func parseSize(s string, defaultVal int64) int64 { //nolint:unused
	// Simple implementation - in production use a proper parser
	s = strings.ToUpper(strings.TrimSpace(s))

	multiplier := int64(1)
	if strings.HasSuffix(s, "G") || strings.HasSuffix(s, "GB") || strings.HasSuffix(s, "GIB") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(s, "IB"), "B"), "G")
	} else if strings.HasSuffix(s, "M") || strings.HasSuffix(s, "MB") || strings.HasSuffix(s, "MIB") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(s, "IB"), "B"), "M")
	}

	var val int64
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return defaultVal
	}

	return val * multiplier
}

// parseInt parses an integer (simplified).
func parseInt(s string, defaultVal int) int { //nolint:unused
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return defaultVal
	}
	return val
}

// PrefetchModules pre-fetches multiple modules in parallel.
func PrefetchModules(ctx context.Context, ociCli OCI, refs []ModuleRef, opts PullOptions) error {
	// Simple parallel fetch - in production use errgroup
	errors := make(chan error, len(refs))

	for _, ref := range refs {
		go func(r ModuleRef) {
			_, err := Pull(ctx, ociCli, r, opts)
			errors <- err
		}(ref)
	}

	// Collect errors
	var firstErr error
	for i := 0; i < len(refs); i++ {
		if err := <-errors; err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

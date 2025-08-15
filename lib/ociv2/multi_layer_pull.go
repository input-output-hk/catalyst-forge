package ociv2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/observability"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/utils"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
)

// PullArtifact pulls a complete artifact with all layers
func (c *client) PullArtifact(ctx context.Context, ref string) (*PullResult, error) {
	operation := "pull_artifact"

	// Validate reference
	if err := utils.ValidateReference(ref); err != nil {
		return nil, observability.NewValidationError(operation, ref, err)
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	// Track operation
	var logger observability.Logger = &observability.NoOpLogger{}
	if c.opts.StructuredLogger != nil {
		logger = c.opts.StructuredLogger
	} else if c.opts.Logger != nil {
		logger = observability.NewDefaultLogger(c.opts.Logger)
	}
	tracker := &observability.OperationTracker{
		StartTime: time.Now(),
		Operation: operation,
		Logger:    logger,
		Fields:    map[string]interface{}{"ref": ref},
	}
	defer func() {
		// Metrics recording handled where durations are known
	}()

	// Normalize reference
	ref = NormalizeRef(ref)

	// Extract registry
	registryHost, err := extractRegistry(ref)
	if err != nil {
		return nil, observability.NewValidationError(operation, ref, fmt.Errorf("failed to extract registry: %w", err))
	}

	// Log operation
	if c.opts.StructuredLogger != nil {
		c.opts.StructuredLogger.Info("pulling artifact", map[string]interface{}{
			"ref": ref,
		})
	} else if c.opts.Logger != nil {
		c.opts.Logger("oci.pull_artifact", "ref", ref)
	}

	// Try ORAS first (handles artifact manifests better)
	result, err := c.pullArtifactORAS(ctx, ref, registryHost)
	if err == nil {
		tracker.Complete(nil, "backend", "oras")
		c.recordMetrics(operation, registryHost, time.Since(tracker.StartTime), nil)
		return result, nil
	}

	// Fallback to ggcr
	result, err2 := c.pullArtifactGGCR(ctx, ref, registryHost)
	if err2 == nil {
		tracker.Complete(nil, "backend", "ggcr", "fallback", true)
		c.recordMetrics(operation, registryHost, time.Since(tracker.StartTime), nil)
		return result, nil
	}

	// Both failed - return the first error
	finalErr := c.wrapError(err, operation, ref, registryHost)
	tracker.Complete(finalErr)
	c.recordMetrics(operation, registryHost, time.Since(tracker.StartTime), finalErr)
	return nil, finalErr
}

// pullArtifactORAS pulls using ORAS
func (c *client) pullArtifactORAS(ctx context.Context, ref, registryHost string) (*PullResult, error) {
	// Create ORAS repository
	repo, err := c.createORASRepo(ctx, ref, registryHost)
	if err != nil {
		return nil, fmt.Errorf("failed to create ORAS repository: %w", err)
	}

	// Create memory store for fetching
	store := memory.New()

	// Fetch manifest and content
	desc, err := oras.Copy(ctx, repo, ref, store, "", oras.DefaultCopyOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from registry: %w", err)
	}

	// Fetch and parse manifest
	manifestData, err := store.Fetch(ctx, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	manifestBytes, err := io.ReadAll(manifestData)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	result := &PullResult{
		Descriptor: Descriptor{
			Ref:       fmt.Sprintf("%s@%s", ref, desc.Digest),
			Digest:    desc.Digest.String(),
			MediaType: desc.MediaType,
			Size:      desc.Size,
		},
		ManifestAnn: make(map[string]string),
		Layers:      []PulledLayer{},
	}

	// Parse based on media type
	switch desc.MediaType {
	case ocispec.MediaTypeImageManifest:
		// OCI image manifest
		var manifest ocispec.Manifest
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse image manifest: %w", err)
		}

		// Extract config if present
		if manifest.Config.Size > 0 {
			configData, err := store.Fetch(ctx, manifest.Config)
			if err == nil {
				result.Config, _ = io.ReadAll(configData)
				result.ConfigMediaType = manifest.Config.MediaType
			}
		}

		// Extract annotations
		if manifest.Annotations != nil {
			result.ManifestAnn = manifest.Annotations
		}

		// Process layers
		for _, layer := range manifest.Layers {
			pulledLayer := c.createPulledLayer(ctx, store, layer, ref, registryHost)
			result.Layers = append(result.Layers, pulledLayer)
		}

	case "application/vnd.oci.artifact.manifest.v1+json":
		// OCI 1.1 artifact manifest
		var manifest ocispec.Manifest
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse artifact manifest: %w", err)
		}

		result.ArtifactType = manifest.ArtifactType

		// Extract config if present
		if manifest.Config.Size > 0 {
			configData, err := store.Fetch(ctx, manifest.Config)
			if err == nil {
				result.Config, _ = io.ReadAll(configData)
				result.ConfigMediaType = manifest.Config.MediaType
			}
		}

		// Extract annotations
		if manifest.Annotations != nil {
			result.ManifestAnn = manifest.Annotations
		}

		// Process layers
		for _, layer := range manifest.Layers {
			pulledLayer := c.createPulledLayer(ctx, store, layer, ref, registryHost)
			result.Layers = append(result.Layers, pulledLayer)
		}

	default:
		return nil, fmt.Errorf("unsupported manifest media type: %s", desc.MediaType)
	}

	return result, nil
}

// pullArtifactGGCR pulls using go-containerregistry
func (c *client) pullArtifactGGCR(ctx context.Context, ref, registryHost string) (*PullResult, error) {
	// Get auth
	authFunc := c.getGGCRAuthFor(registryHost)

	// Parse reference
	nameRef, err := parseGGCRRef(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference: %w", err)
	}

	// Set up remote options
	remoteOpts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithUserAgent(c.opts.UserAgent),
	}
	if authFunc != nil {
		auth, err := authFunc()
		if err == nil && auth != nil {
			remoteOpts = append(remoteOpts, remote.WithAuth(auth))
		}
	}
	if c.opts.PlainHTTP {
		remoteOpts = append(remoteOpts, remote.WithTransport(c.transport))
	}

	// Get descriptor first
	desc, err := remote.Get(nameRef, remoteOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to get descriptor: %w", err)
	}

	result := &PullResult{
		Descriptor: Descriptor{
			Ref:       fmt.Sprintf("%s@%s", ref, desc.Digest),
			Digest:    desc.Digest.String(),
			MediaType: string(desc.MediaType),
			Size:      desc.Size,
		},
		ManifestAnn: make(map[string]string),
		Layers:      []PulledLayer{},
	}

	// Get the image/index
	switch desc.MediaType {
	case types.OCIManifestSchema1, types.DockerManifestSchema2:
		// Pull as image
		img, err := desc.Image()
		if err != nil {
			return nil, fmt.Errorf("failed to get image: %w", err)
		}

		// Get config
		configFile, err := img.ConfigFile()
		if err == nil && configFile != nil {
			configData, _ := json.Marshal(configFile)
			result.Config = configData
			result.ConfigMediaType = "application/vnd.oci.image.config.v1+json"
		}

		// Get manifest
		manifest, err := img.Manifest()
		if err == nil && manifest != nil {
			if manifest.Annotations != nil {
				result.ManifestAnn = manifest.Annotations
			}
		}

		// Get layers
		layers, err := img.Layers()
		if err != nil {
			return nil, fmt.Errorf("failed to get layers: %w", err)
		}

		for i, layer := range layers {
			digest, _ := layer.Digest()
			size, _ := layer.Size()
			mediaType, _ := layer.MediaType()

			// Create pulled layer with lazy loading
			pulledLayer := PulledLayer{
				MediaType: string(mediaType),
				Size:      size,
				Digest:    digest.String(),
				Open: func() (io.ReadCloser, error) {
					return layer.Compressed()
				},
			}

			// Add annotations from manifest if available
			if manifest != nil && i < len(manifest.Layers) {
				pulledLayer.Annotations = manifest.Layers[i].Annotations
			}

			result.Layers = append(result.Layers, pulledLayer)
		}

	default:
		return nil, fmt.Errorf("unsupported media type: %s", desc.MediaType)
	}

	return result, nil
}

// createPulledLayer creates a PulledLayer with lazy loading from ORAS store
func (c *client) createPulledLayer(ctx context.Context, store oras.Target, desc ocispec.Descriptor, ref, registryHost string) PulledLayer {
	return PulledLayer{
		MediaType:   desc.MediaType,
		Size:        desc.Size,
		Digest:      desc.Digest.String(),
		Annotations: desc.Annotations,
		Open: func() (io.ReadCloser, error) {
			// Fetch from store or re-fetch from registry
			rc, err := store.Fetch(ctx, desc)
			if err != nil {
				// Try to re-fetch from registry
				return c.fetchLayerFromRegistry(ctx, ref, desc.Digest.String(), registryHost)
			}
			return rc, nil
		},
	}
}

// fetchLayerFromRegistry fetches a specific layer from the registry
func (c *client) fetchLayerFromRegistry(ctx context.Context, ref, digestStr, registryHost string) (io.ReadCloser, error) {
	// This is a fallback method to fetch a layer directly by digest
	layerRef := fmt.Sprintf("%s@%s", ref, digestStr)

	// Try ORAS first
	repo, err := c.createORASRepo(ctx, layerRef, registryHost)
	if err == nil {
		desc := ocispec.Descriptor{Digest: digest.Digest(digestStr)}
		rc, err := repo.Fetch(ctx, desc)
		if err == nil {
			return rc, nil
		}
	}

	// Fallback to ggcr
	authFunc := c.getGGCRAuthFor(registryHost)
	nameRef, err := parseGGCRRef(layerRef)
	if err != nil {
		return nil, err
	}

	remoteOpts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithUserAgent(c.opts.UserAgent),
	}
	if authFunc != nil {
		auth, err := authFunc()
		if err == nil && auth != nil {
			remoteOpts = append(remoteOpts, remote.WithAuth(auth))
		}
	}

	// Convert to digest reference
	digestRef, ok := nameRef.(name.Digest)
	if !ok {
		return nil, fmt.Errorf("layer ref must be a digest reference")
	}

	layer, err := remote.Layer(digestRef, remoteOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch layer: %w", err)
	}

	return layer.Compressed()
}

// parseGGCRRef parses a reference for go-containerregistry
func parseGGCRRef(ref string) (name.Reference, error) {
	// Remove oci:// prefix if present
	ref = NormalizeRef(ref)

	// Parse as tag or digest
	if IsDigestRef(ref) {
		return name.ParseReference(ref)
	}

	// Default to latest tag if no tag specified
	if !strings.Contains(ref, ":") {
		ref = ref + ":latest"
	}

	return name.ParseReference(ref)
}

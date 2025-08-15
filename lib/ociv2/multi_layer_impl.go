package ociv2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/ociv2/observability"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/utils"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// PushArtifact pushes a multi-layer artifact with the specified options
func (c *client) PushArtifact(ctx context.Context, ref string, opts PackOptions) (Descriptor, error) {
	operation := "push_artifact"

	// Acquire semaphore for concurrency control
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return Descriptor{}, ctx.Err()
	}

	// Validate options
	if err := opts.Validate(); err != nil {
		return Descriptor{}, observability.NewValidationError(operation, ref, err)
	}

	// Validate reference
	if err := utils.ValidateReference(ref); err != nil {
		return Descriptor{}, observability.NewValidationError(operation, ref, err)
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
		Fields:    map[string]interface{}{"ref": ref, "layers": len(opts.Layers)},
	}
	defer func() {
		// Metrics recording handled where durations are known
	}()

	// Normalize reference
	ref = NormalizeRef(ref)

	// Extract registry
	registry, err := extractRegistry(ref)
	if err != nil {
		return Descriptor{}, observability.NewValidationError(operation, ref, fmt.Errorf("failed to extract registry: %w", err))
	}

	// Add standard annotations
	if opts.ManifestAnnotations == nil {
		opts.ManifestAnnotations = make(map[string]string)
	}
	if _, ok := opts.ManifestAnnotations[utils.AnnCreated]; !ok {
		opts.ManifestAnnotations[utils.AnnCreated] = time.Now().UTC().Format(time.RFC3339)
	}

	// Log operation
	if c.opts.StructuredLogger != nil {
		c.opts.StructuredLogger.Info("pushing multi-layer artifact", map[string]interface{}{
			"ref":          ref,
			"layers":       len(opts.Layers),
			"artifactType": opts.ArtifactType,
			"hasConfig":    len(opts.Config) > 0,
		})
	} else if c.opts.Logger != nil {
		c.opts.Logger("oci.push_artifact", "ref", ref, "layers", len(opts.Layers))
	}

	// Set defaults
	if !opts.PreferArtifactManifest && !opts.FallbackImageManifest {
		opts.PreferArtifactManifest = true
		opts.FallbackImageManifest = true
	}

	// Try artifact manifest first if preferred
	if opts.PreferArtifactManifest {
		desc, err := c.pushArtifactManifest(ctx, ref, registry, opts)
		if err == nil {
			tracker.Complete(nil, "manifest", "artifact")
			c.recordMetrics(operation, registry, time.Since(tracker.StartTime), nil)
			return desc, nil
		}

		// Check if we should fallback
		if opts.FallbackImageManifest && c.shouldFallbackToImage(err) {
			if c.opts.StructuredLogger != nil {
				c.opts.StructuredLogger.Debug("falling back to image manifest", map[string]interface{}{
					"ref":   ref,
					"error": err.Error(),
				})
			} else if c.opts.Logger != nil {
				c.opts.Logger("oci.push_artifact.fallback", "ref", ref, "error", err.Error())
			}
		} else {
			// No fallback or error is not recoverable
			finalErr := c.wrapError(err, operation, ref, registry)
			tracker.Complete(finalErr)
			c.recordMetrics(operation, registry, time.Since(tracker.StartTime), finalErr)
			return Descriptor{}, finalErr
		}
	}

	// Fallback to image manifest
	desc, err := c.pushImageManifest(ctx, ref, registry, opts)
	if err != nil {
		finalErr := c.wrapError(err, operation, ref, registry)
		tracker.Complete(finalErr)
		c.recordMetrics(operation, registry, time.Since(tracker.StartTime), finalErr)
		return Descriptor{}, finalErr
	}

	tracker.Complete(nil, "manifest", "image", "fallback", true)
	c.recordMetrics(operation, registry, time.Since(tracker.StartTime), nil)
	return desc, nil
}

// pushArtifactManifest pushes using OCI 1.1 artifact manifest
func (c *client) pushArtifactManifest(ctx context.Context, ref, registry string, opts PackOptions) (Descriptor, error) {
	// Create ORAS repository
	repo, err := c.createORASRepo(ctx, ref, registry)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to create ORAS repository: %w", err)
	}

	// Create memory store for staging
	store := memory.New()

	// Build layers
	var layers []ocispec.Descriptor
	for i, layer := range opts.Layers {
		// Read layer content
		data, err := io.ReadAll(layer.Reader)
		if err != nil {
			return Descriptor{}, fmt.Errorf("failed to read layer %d: %w", i, err)
		}

		// Compute digest
		dgst := sha256.Sum256(data)
		digestStr := "sha256:" + hex.EncodeToString(dgst[:])

		// Create descriptor
		desc := ocispec.Descriptor{
			MediaType: layer.MediaType,
			Digest:    digest.Digest(digestStr),
			Size:      int64(len(data)),
		}

		// Add layer annotations
		if layer.Title != "" || len(layer.Annotations) > 0 {
			desc.Annotations = make(map[string]string)
			if layer.Title != "" {
				desc.Annotations[utils.AnnTitle] = layer.Title
			}
			for k, v := range layer.Annotations {
				desc.Annotations[k] = v
			}
		}

		// Push to memory store
		if err := store.Push(ctx, desc, bytes.NewReader(data)); err != nil {
			return Descriptor{}, fmt.Errorf("failed to stage layer %d: %w", i, err)
		}

		layers = append(layers, desc)
	}

	// Handle config if present
	var configDesc *ocispec.Descriptor
	if len(opts.Config) > 0 {
		dgst := sha256.Sum256(opts.Config)
		digestStr := "sha256:" + hex.EncodeToString(dgst[:])

		configDesc = &ocispec.Descriptor{
			MediaType: opts.ConfigMediaType,
			Digest:    digest.Digest(digestStr),
			Size:      int64(len(opts.Config)),
		}

		// Push config to memory store
		if err := store.Push(ctx, *configDesc, bytes.NewReader(opts.Config)); err != nil {
			return Descriptor{}, fmt.Errorf("failed to stage config: %w", err)
		}
	}

	// Pack manifest using ORAS Pack with annotations
	packOpts := oras.PackManifestOptions{
		Subject:             nil, // No subject for now
		ConfigDescriptor:    configDesc,
		Layers:              layers,
		ManifestAnnotations: opts.ManifestAnnotations,
	}

	// Create manifest
	root, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, opts.ArtifactType, packOpts)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to pack manifest: %w", err)
	}

	// Tag the manifest in the store
	if err := store.Tag(ctx, root, ref); err != nil {
		return Descriptor{}, fmt.Errorf("failed to tag manifest: %w", err)
	}

	// Copy from memory store to registry
	_, err = oras.Copy(ctx, store, ref, repo, ref, oras.DefaultCopyOptions)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to push to registry: %w", err)
	}

	// Get canonical reference
	canonicalRef := fmt.Sprintf("%s@%s", ref, root.Digest)

	return Descriptor{
		Ref:       canonicalRef,
		Digest:    root.Digest.String(),
		MediaType: root.MediaType,
		Size:      root.Size,
	}, nil
}

// pushImageManifest pushes using OCI image manifest as fallback
func (c *client) pushImageManifest(ctx context.Context, ref, _ string, opts PackOptions) (Descriptor, error) {
	// For image manifest fallback, we need to restructure the data
	// Use the first layer as the main layer, config as config

	if len(opts.Layers) == 0 {
		return Descriptor{}, fmt.Errorf("at least one layer required for image manifest")
	}

	// If no config provided, create an empty one
	config := opts.Config
	configMT := opts.ConfigMediaType
	if len(config) == 0 {
		config = []byte("{}")
		configMT = "application/vnd.oci.image.config.v1+json"
	}

	// For simplicity, push the first layer as the main content
	// Additional layers can be added as separate pushes or combined
	firstLayer := opts.Layers[0]

	// Read first layer
	data, err := io.ReadAll(firstLayer.Reader)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to read layer: %w", err)
	}

	// Use existing ggcr push logic
	desc, err := c.ggcrPushConfigAndLayer(ctx, ref, config, configMT,
		bytes.NewReader(data), int64(len(data)), firstLayer.MediaType, opts.ManifestAnnotations)
	if err != nil {
		return Descriptor{}, err
	}

	// Convert from ocispec.Descriptor to our Descriptor
	canonicalRef := ref
	if !strings.Contains(ref, "@") {
		canonicalRef = fmt.Sprintf("%s@%s", ref, desc.Digest)
	}

	return Descriptor{
		Ref:       canonicalRef,
		Digest:    desc.Digest.String(),
		MediaType: desc.MediaType,
		Size:      desc.Size,
	}, nil
}

// createORASRepo creates an ORAS repository client
func (c *client) createORASRepo(_ context.Context, ref, _ string) (oras.Target, error) {
	// Create repository
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Configure plain HTTP if needed
	if c.opts.PlainHTTP || isLoopbackRegistry(ref) {
		repo.PlainHTTP = true
	}

	// Set up auth
	authFunc := c.getORASAuth()
	if authFunc != nil {
		repo.Client = &auth.Client{
			Credential: authFunc,
		}
	}

	return repo, nil
}

// shouldFallbackToImage determines if we should fallback to image manifest
func (c *client) shouldFallbackToImage(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error patterns indicating artifact manifest rejection
	errStr := err.Error()

	// Common patterns for registries that don't support artifact manifests
	patterns := []string{
		"unsupported media type",
		"unknown media type",
		"not supported",
		"400 Bad Request",
		"415 Unsupported Media Type",
		"501 Not Implemented",
		"manifest invalid",
	}

	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}

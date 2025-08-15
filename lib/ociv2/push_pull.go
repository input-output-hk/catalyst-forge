package ociv2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/auth"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/internal"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/observability"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/utils"
	orasauth "oras.land/oras-go/v2/registry/remote/auth"
)

// -------- Generic primitives --------

// Resolve fetches the manifest and returns a complete descriptor
func (c *client) Resolve(ctx context.Context, ref string) (Descriptor, error) {
	return c.headOrResolve(ctx, ref, true)
}

// Head performs a HEAD request to get descriptor metadata
func (c *client) Head(ctx context.Context, ref string) (Descriptor, error) {
	return c.headOrResolve(ctx, ref, false)
}

// headOrResolve is the internal helper for both Head and Resolve
func (c *client) headOrResolve(ctx context.Context, ref string, fullResolve bool) (Descriptor, error) {
	operation := "head"
	if fullResolve {
		operation = "resolve"
	}
	
	// Validate reference format
	if err := utils.ValidateReference(ref); err != nil {
		return Descriptor{}, observability.NewValidationError(operation, ref, err)
	}
	
	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()
	
	// Normalize reference
	ref = NormalizeRef(ref)
	
	// Extract registry
	registry, err := extractRegistry(ref)
	if err != nil {
		return Descriptor{}, observability.NewValidationError(operation, ref, fmt.Errorf("failed to extract registry: %w", err))
	}
	
	// Create operation tracker
	var logger Logger = &observability.NoOpLogger{}
	if c.opts.StructuredLogger != nil {
		logger = c.opts.StructuredLogger
	} else if c.opts.Logger != nil {
		logger = observability.NewDefaultLogger(c.opts.Logger)
	}
	
	tracker := observability.NewOperationTracker(logger, operation, ref, registry)
	tracker.Start("fetching artifact metadata")
	defer func() {
		if r := recover(); r != nil {
			tracker.Complete(fmt.Errorf("panic: %v", r))
			panic(r)
		}
	}()
	
	// Try ORAS first if preferred
	if c.opts.PreferArtifactManifest {
		desc, err := c.orasHeadOrResolve(ctx, ref, registry, fullResolve)
		if err == nil {
			result := c.descriptorFromOCISpec(desc, ref)
			tracker.Complete(nil, "backend", "oras")
			c.recordMetrics(operation, registry, time.Since(tracker.StartTime), nil)
			return result, nil
		}
		
		// If fallback is enabled, try ggcr
		if c.opts.FallbackImageManifest {
			desc2, err2 := c.ggcrHeadOrResolve(ctx, ref, fullResolve)
			if err2 == nil {
				result := c.descriptorFromOCISpec(desc2, ref)
				tracker.Complete(nil, "backend", "ggcr", "fallback", true)
				c.recordMetrics(operation, registry, time.Since(tracker.StartTime), nil)
				return result, nil
			}
		}
		
		// Both failed
		finalErr := c.wrapError(err, operation, ref, registry)
		tracker.Complete(finalErr)
		c.recordMetrics(operation, registry, time.Since(tracker.StartTime), finalErr)
		return Descriptor{}, finalErr
	}
	
	// Try ggcr first
	desc, err := c.ggcrHeadOrResolve(ctx, ref, fullResolve)
	if err == nil {
		result := c.descriptorFromOCISpec(desc, ref)
		tracker.Complete(nil, "backend", "ggcr")
		c.recordMetrics(operation, registry, time.Since(tracker.StartTime), nil)
		return result, nil
	}
	
	// Fallback to ORAS
	desc2, err2 := c.orasHeadOrResolve(ctx, ref, registry, fullResolve)
	if err2 == nil {
		result := c.descriptorFromOCISpec(desc2, ref)
		tracker.Complete(nil, "backend", "oras", "fallback", true)
		c.recordMetrics(operation, registry, time.Since(tracker.StartTime), nil)
		return result, nil
	}
	
	// Both failed
	finalErr := c.wrapError(err, operation, ref, registry)
	tracker.Complete(finalErr)
	c.recordMetrics(operation, registry, time.Since(tracker.StartTime), finalErr)
	return Descriptor{}, finalErr
}

// -------- JSON operations --------

// PushJSON pushes a JSON blob as an artifact
func (c *client) PushJSON(ctx context.Context, ref string, mediaType string, payload []byte, ann Annotations) (Descriptor, error) {
	operation := "push_json"
	
	// Acquire semaphore for concurrency control
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return Descriptor{}, ctx.Err()
	}
	
	// Validate inputs
	if err := utils.ValidateReference(ref); err != nil {
		return Descriptor{}, observability.NewValidationError(operation, ref, err)
	}
	
	if err := utils.ValidateMediaType(mediaType); err != nil {
		return Descriptor{}, observability.NewValidationError(operation, ref, err)
	}
	
	if err := utils.ValidateBlobSize(int64(len(payload)), c.opts.MaxBlobSize); err != nil {
		return Descriptor{}, observability.NewValidationError(operation, ref, err)
	}
	
	if ann != nil {
		// Convert to utils.Annotations for validation
		utilsAnn := utils.Annotations(ann)
		if err := utils.ValidateAnnotations(utilsAnn); err != nil {
			return Descriptor{}, observability.NewValidationError(operation, ref, err)
		}
	}
	
	// Continue with original validation
	if err := validateRef(ref); err != nil {
		return Descriptor{}, observability.NewValidationError(operation, ref, err)
	}
	
	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()
	
	// Normalize reference
	ref = NormalizeRef(ref)
	
	// Extract registry
	registry, err := extractRegistry(ref)
	if err != nil {
		return Descriptor{}, observability.NewValidationError(operation, ref, fmt.Errorf("failed to extract registry: %w", err))
	}
	
	// Create operation tracker
	var logger Logger = &observability.NoOpLogger{}
	if c.opts.StructuredLogger != nil {
		logger = c.opts.StructuredLogger
	} else if c.opts.Logger != nil {
		logger = observability.NewDefaultLogger(c.opts.Logger)
	}
	
	tracker := observability.NewOperationTracker(logger, operation, ref, registry)
	tracker.WithField("media_type", mediaType).WithField("payload_size", len(payload))
	tracker.Start("pushing JSON artifact")
	defer func() {
		if r := recover(); r != nil {
			tracker.Complete(fmt.Errorf("panic: %v", r))
			panic(r)
		}
	}()
	
	// Add standard annotations
	if ann == nil {
		ann = Annotations(utils.NewAnnotations())
	} else {
		ann = Annotations(utils.NewAnnotations().Merge(utils.Annotations(ann)))
	}
	
	// Detect registry type for intelligent fallback
	registryType := internal.DetectRegistryType(registry)
	tracker.WithField("registry_type", registryType.String())
	
	// Try artifact manifest first via ORAS
	if c.opts.PreferArtifactManifest {
		desc, err := c.orasPushConfigOnly(ctx, ref, registry, mediaType, payload, ann)
		if err == nil {
			result := ensureDigest(ref, c.descriptorFromOCISpec(desc, ref))
			tracker.Complete(nil, "backend", "oras", "manifest_type", "artifact")
			c.recordMetrics(operation, registry, time.Since(tracker.StartTime), nil)
			return result, nil
		}
		
		// Check if this error suggests we should fallback to image manifest
		if c.opts.FallbackImageManifest && internal.ShouldFallbackToImageManifest(err, registryType) {
			logger.Info("Falling back to image manifest", "reason", err.Error())
			c.recordFallbackAttempt(false, false) // Neither succeeded yet
			
			desc2, err2 := c.ggcrPushJSONLayer(ctx, ref, mediaType, payload, ann)
			if err2 == nil {
				result := ensureDigest(ref, c.descriptorFromOCISpec(desc2, ref))
				tracker.Complete(nil, "backend", "ggcr", "manifest_type", "image", "fallback", true)
				c.recordMetrics(operation, registry, time.Since(tracker.StartTime), nil)
				c.recordFallbackAttempt(false, true) // Image succeeded
				return result, nil
			}
			
			// Both methods failed
			finalErr := observability.NewFallbackError(operation, ref, err, err2)
			tracker.Complete(finalErr, "artifact_error", err.Error(), "image_error", err2.Error())
			c.recordMetrics(operation, registry, time.Since(tracker.StartTime), finalErr)
			c.recordFallbackAttempt(false, false) // Both failed
			return Descriptor{}, finalErr
		}
		
		// No fallback enabled or not applicable
		finalErr := c.wrapError(err, operation, ref, registry)
		tracker.Complete(finalErr)
		c.recordMetrics(operation, registry, time.Since(tracker.StartTime), finalErr)
		return Descriptor{}, finalErr
	}
	
	// Use image manifest directly
	desc, err := c.ggcrPushJSONLayer(ctx, ref, mediaType, payload, ann)
	if err == nil {
		return ensureDigest(ref, c.descriptorFromOCISpec(desc, ref)), nil
	}
	
	return Descriptor{}, err
}

// PullJSON pulls a JSON blob artifact
func (c *client) PullJSON(ctx context.Context, ref string, wantMediaType string) ([]byte, Descriptor, error) {
	// Validate reference
	if err := validateRef(ref); err != nil {
		return nil, Descriptor{}, err
	}
	
	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()
	
	// Normalize reference
	ref = NormalizeRef(ref)
	
	// Extract registry
	registry, err := extractRegistry(ref)
	if err != nil {
		return nil, Descriptor{}, fmt.Errorf("failed to extract registry: %w", err)
	}
	
	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.pull.json", "ref", ref, "wantMediaType", wantMediaType)
	}
	
	// Try ORAS first
	data, desc, err := c.orasPullConfig(ctx, ref, registry, wantMediaType)
	if err == nil {
		return data, ensureDigest(ref, c.descriptorFromOCISpec(desc, ref)), nil
	}
	
	// Fallback to ggcr
	data2, desc2, err2 := c.ggcrPullJSONLayer(ctx, ref, wantMediaType)
	if err2 == nil {
		return data2, ensureDigest(ref, c.descriptorFromOCISpec(desc2, ref)), nil
	}
	
	// Return the first error if both failed
	return nil, Descriptor{}, err
}

// -------- TAR operations --------

// PushTar pushes a tar stream with a JSON config
func (c *client) PushTar(ctx context.Context, ref string, cfg []byte, cfgMT, layerMT string, tar io.Reader, size int64, ann Annotations) (Descriptor, error) {
	// Acquire semaphore for concurrency control
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return Descriptor{}, ctx.Err()
	}
	
	// Validate inputs
	if err := utils.ValidateReference(ref); err != nil {
		return Descriptor{}, observability.NewValidationError("push_tar", ref, err)
	}
	
	if err := utils.ValidateMediaType(cfgMT); err != nil {
		return Descriptor{}, observability.NewValidationError("push_tar", ref, err)
	}
	
	if err := utils.ValidateMediaType(layerMT); err != nil {
		return Descriptor{}, observability.NewValidationError("push_tar", ref, err)
	}
	
	if err := utils.ValidateBlobSize(size, c.opts.MaxBlobSize); err != nil {
		return Descriptor{}, observability.NewValidationError("push_tar", ref, err)
	}
	
	// Wrap reader with buffer for efficient streaming
	if c.opts.StreamBufferSize > 0 {
		tar = &utils.BufferedReader{
			R:   tar,
			Buf: make([]byte, c.opts.StreamBufferSize),
		}
	}
	
	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()
	
	// Normalize reference
	ref = NormalizeRef(ref)
	
	// Extract registry
	registry, err := extractRegistry(ref)
	if err != nil {
		return Descriptor{}, fmt.Errorf("failed to extract registry: %w", err)
	}
	
	// Add standard annotations
	if ann == nil {
		ann = Annotations(utils.NewAnnotations())
	} else {
		ann = Annotations(utils.NewAnnotations().Merge(utils.Annotations(ann)))
	}
	
	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.push.tar", "ref", ref, "cfgMT", cfgMT, "layerMT", layerMT, "size", size)
	}
	
	// Detect registry type for intelligent fallback
	registryType := internal.DetectRegistryType(registry)
	
	// Try artifact manifest first via ORAS
	if c.opts.PreferArtifactManifest {
		desc, err := c.orasPushConfigAndLayer(ctx, ref, registry, cfg, cfgMT, tar, size, layerMT, ann)
		if err == nil {
			if c.opts.Logger != nil {
				c.opts.Logger("oci.push.tar.artifact.success", "ref", ref, "registry_type", registryType.String())
			}
			return ensureDigest(ref, c.descriptorFromOCISpec(desc, ref)), nil
		}
		
		// Check if this error suggests we should fallback to image manifest
		if c.opts.FallbackImageManifest && internal.ShouldFallbackToImageManifest(err, registryType) {
			if c.opts.Logger != nil {
				c.opts.Logger("oci.push.tar.fallback", "ref", ref, "registry_type", registryType.String(), "reason", err.Error())
			}
			
			desc2, err2 := c.ggcrPushConfigAndLayer(ctx, ref, cfg, cfgMT, tar, size, layerMT, ann)
			if err2 == nil {
				if c.opts.Logger != nil {
					c.opts.Logger("oci.push.tar.image.success", "ref", ref, "registry_type", registryType.String())
				}
				return ensureDigest(ref, c.descriptorFromOCISpec(desc2, ref)), nil
			}
			
			// Both methods failed, return the more informative error
			if c.opts.Logger != nil {
				c.opts.Logger("oci.push.tar.failed", "ref", ref, "artifact_err", err.Error(), "image_err", err2.Error())
			}
			return Descriptor{}, fmt.Errorf("push failed with both artifact manifest (%v) and image manifest (%v)", err, err2)
		}
		
		return Descriptor{}, err
	}
	
	// Use image manifest directly
	desc, err := c.ggcrPushConfigAndLayer(ctx, ref, cfg, cfgMT, tar, size, layerMT, ann)
	if err == nil {
		return ensureDigest(ref, c.descriptorFromOCISpec(desc, ref)), nil
	}
	
	return Descriptor{}, err
}

// PullTar pulls a tar layer from an artifact
func (c *client) PullTar(ctx context.Context, ref string, layerMT string) (io.ReadCloser, Descriptor, error) {
	// Validate reference
	if err := validateRef(ref); err != nil {
		return nil, Descriptor{}, err
	}
	
	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()
	
	// Normalize reference
	ref = NormalizeRef(ref)
	
	// Extract registry
	registry, err := extractRegistry(ref)
	if err != nil {
		return nil, Descriptor{}, fmt.Errorf("failed to extract registry: %w", err)
	}
	
	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.pull.tar", "ref", ref, "layerMT", layerMT)
	}
	
	// Try ORAS first
	rc, desc, err := c.orasPullLayer(ctx, ref, registry, layerMT)
	if err == nil {
		return rc, ensureDigest(ref, c.descriptorFromOCISpec(desc, ref)), nil
	}
	
	// Fallback to ggcr
	rc2, desc2, err2 := c.ggcrPullLayer(ctx, ref, layerMT)
	if err2 == nil {
		return rc2, ensureDigest(ref, c.descriptorFromOCISpec(desc2, ref)), nil
	}
	
	// Return the first error if both failed
	return nil, Descriptor{}, err
}

// -------- Release Bundle helpers --------

// PushReleaseBundle pushes a Release Bundle JSON
func (c *client) PushReleaseBundle(ctx context.Context, ref string, releaseJSON []byte, ann Annotations) (Descriptor, error) {
	// Ensure annotations
	if ann == nil {
		ann = Annotations{}
	}
	
	// Add Forge kind
	ann[utils.AnnForgeKind] = "release"
	
	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.push.release", "ref", ref, "size", len(releaseJSON))
	}
	
	return c.PushJSON(ctx, ref, MTReleaseConfig, releaseJSON, ann)
}

// PullReleaseBundle pulls a Release Bundle JSON
func (c *client) PullReleaseBundle(ctx context.Context, ref string) ([]byte, Descriptor, error) {
	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.pull.release", "ref", ref)
	}
	
	return c.PullJSON(ctx, ref, MTReleaseConfig)
}

// -------- Rendered Set helpers --------

// PushRenderedSet pushes a Rendered Set (index + tar)
func (c *client) PushRenderedSet(ctx context.Context, ref string, indexJSON []byte, tar io.Reader, size int64, ann Annotations) (Descriptor, error) {
	// Ensure annotations
	if ann == nil {
		ann = Annotations{}
	}
	
	// Add Forge kind
	ann[utils.AnnForgeKind] = "rendered"
	
	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.push.rendered", "ref", ref, "indexSize", len(indexJSON), "tarSize", size)
	}
	
	return c.PushTar(ctx, ref, indexJSON, MTRenderedIndex, MTRenderedTarGz, tar, size, ann)
}

// PullRenderedSet pulls a Rendered Set (returns tar stream, index JSON, and descriptor)
func (c *client) PullRenderedSet(ctx context.Context, ref string) (io.ReadCloser, []byte, Descriptor, error) {
	// Log operation
	if c.opts.Logger != nil {
		c.opts.Logger("oci.pull.rendered", "ref", ref)
	}
	
	// First pull the index (config)
	indexJSON, desc1, err := c.PullJSON(ctx, ref, MTRenderedIndex)
	if err != nil {
		return nil, nil, Descriptor{}, fmt.Errorf("failed to pull rendered index: %w", err)
	}
	
	// Then pull the tar layer
	rc, desc2, err := c.PullTar(ctx, ref, MTRenderedTarGz)
	if err != nil {
		return nil, nil, Descriptor{}, fmt.Errorf("failed to pull rendered tar: %w", err)
	}
	
	// Return the tar stream, index, and the manifest descriptor (prefer desc2 if it has digest)
	if desc2.Digest != "" {
		return rc, indexJSON, desc2, nil
	}
	return rc, indexJSON, desc1, nil
}

// -------- Internal helpers for backend operations --------

// orasHeadOrResolve uses ORAS to get descriptor
func (c *client) orasHeadOrResolve(ctx context.Context, ref string, registry string, fullResolve bool) (*ocispec.Descriptor, error) {
	// Get auth
	authFunc := c.getORASAuth()
	
	// Create ORAS client
	orasClient := internal.NewORASClient(
		c.opts.PlainHTTP || isLocalRegistry(ref),
		c.opts.UserAgent,
		authFunc,
	)
	
	if fullResolve {
		return orasClient.Resolve(ctx, ref, registry)
	}
	return orasClient.Head(ctx, ref, registry)
}

// ggcrHeadOrResolve uses ggcr to get descriptor
func (c *client) ggcrHeadOrResolve(ctx context.Context, ref string, fullResolve bool) (*ocispec.Descriptor, error) {
	// Get auth
	authFunc := c.getGGCRAuth()
	
	// Create GGCR client
	ggcrClient := internal.NewGGCRClient(
		c.opts.PlainHTTP || isLocalRegistry(ref),
		c.opts.UserAgent,
		authFunc,
		nil, // Use default HTTP client
	)
	
	if fullResolve {
		return ggcrClient.Resolve(ctx, ref)
	}
	return ggcrClient.Head(ctx, ref)
}

// orasPushConfigOnly pushes JSON as artifact config
func (c *client) orasPushConfigOnly(ctx context.Context, ref string, registry string, mediaType string, payload []byte, ann Annotations) (*ocispec.Descriptor, error) {
	// Get auth
	authFunc := c.getORASAuth()
	
	// Create ORAS client
	orasClient := internal.NewORASClient(
		c.opts.PlainHTTP || isLocalRegistry(ref),
		c.opts.UserAgent,
		authFunc,
	)
	
	// Convert annotations to map[string]string
	annotations := make(map[string]string)
	for k, v := range ann {
		annotations[k] = v
	}
	
	return orasClient.PushConfigOnly(ctx, ref, registry, mediaType, payload, annotations)
}

// orasPushConfigAndLayer pushes config + tar layer
func (c *client) orasPushConfigAndLayer(ctx context.Context, ref string, registry string, cfg []byte, cfgMT string, tar io.Reader, size int64, layerMT string, ann Annotations) (*ocispec.Descriptor, error) {
	// Get auth
	authFunc := c.getORASAuth()
	
	// Create ORAS client
	orasClient := internal.NewORASClient(
		c.opts.PlainHTTP || isLocalRegistry(ref),
		c.opts.UserAgent,
		authFunc,
	)
	
	// Convert annotations to map[string]string
	annotations := make(map[string]string)
	for k, v := range ann {
		annotations[k] = v
	}
	
	return orasClient.PushConfigAndLayer(ctx, ref, registry, cfg, cfgMT, tar, size, layerMT, annotations)
}

// orasPullConfig pulls artifact config
func (c *client) orasPullConfig(ctx context.Context, ref string, registry string, wantMT string) ([]byte, *ocispec.Descriptor, error) {
	// Get auth
	authFunc := c.getORASAuth()
	
	// Create ORAS client
	orasClient := internal.NewORASClient(
		c.opts.PlainHTTP || isLocalRegistry(ref),
		c.opts.UserAgent,
		authFunc,
	)
	
	return orasClient.PullConfig(ctx, ref, registry, wantMT)
}

// orasPullLayer pulls artifact layer
func (c *client) orasPullLayer(ctx context.Context, ref string, registry string, layerMT string) (io.ReadCloser, *ocispec.Descriptor, error) {
	// Get auth
	authFunc := c.getORASAuth()
	
	// Create ORAS client
	orasClient := internal.NewORASClient(
		c.opts.PlainHTTP || isLocalRegistry(ref),
		c.opts.UserAgent,
		authFunc,
	)
	
	return orasClient.PullLayer(ctx, ref, registry, layerMT)
}

// ggcrPushJSONLayer pushes JSON as image layer
func (c *client) ggcrPushJSONLayer(ctx context.Context, ref string, mediaType string, payload []byte, ann Annotations) (*ocispec.Descriptor, error) {
	// Get auth
	authFunc := c.getGGCRAuth()
	
	// Create GGCR client
	ggcrClient := internal.NewGGCRClient(
		c.opts.PlainHTTP || isLocalRegistry(ref),
		c.opts.UserAgent,
		authFunc,
		nil, // Use default HTTP client
	)
	
	// Convert annotations to map[string]string
	annotations := make(map[string]string)
	for k, v := range ann {
		annotations[k] = v
	}
	
	return ggcrClient.PushJSONLayer(ctx, ref, mediaType, payload, annotations)
}

// ggcrPushConfigAndLayer pushes config + tar layer as image
func (c *client) ggcrPushConfigAndLayer(ctx context.Context, ref string, cfg []byte, cfgMT string, tar io.Reader, size int64, layerMT string, ann Annotations) (*ocispec.Descriptor, error) {
	// Get auth
	authFunc := c.getGGCRAuth()
	
	// Create GGCR client
	ggcrClient := internal.NewGGCRClient(
		c.opts.PlainHTTP || isLocalRegistry(ref),
		c.opts.UserAgent,
		authFunc,
		nil, // Use default HTTP client
	)
	
	// Convert annotations to map[string]string
	annotations := make(map[string]string)
	for k, v := range ann {
		annotations[k] = v
	}
	
	return ggcrClient.PushConfigAndLayer(ctx, ref, cfg, cfgMT, tar, size, layerMT, annotations)
}

// ggcrPullJSONLayer pulls JSON from image layer
func (c *client) ggcrPullJSONLayer(ctx context.Context, ref string, wantMT string) ([]byte, *ocispec.Descriptor, error) {
	// Get auth
	authFunc := c.getGGCRAuth()
	
	// Create GGCR client
	ggcrClient := internal.NewGGCRClient(
		c.opts.PlainHTTP || isLocalRegistry(ref),
		c.opts.UserAgent,
		authFunc,
		nil, // Use default HTTP client
	)
	
	return ggcrClient.PullJSONLayer(ctx, ref, wantMT)
}

// ggcrPullLayer pulls tar from image layer
func (c *client) ggcrPullLayer(ctx context.Context, ref string, layerMT string) (io.ReadCloser, *ocispec.Descriptor, error) {
	// Get auth
	authFunc := c.getGGCRAuth()
	
	// Create GGCR client
	ggcrClient := internal.NewGGCRClient(
		c.opts.PlainHTTP || isLocalRegistry(ref),
		c.opts.UserAgent,
		authFunc,
		nil, // Use default HTTP client
	)
	
	return ggcrClient.PullLayer(ctx, ref, layerMT)
}

// getORASAuth returns auth function for ORAS
func (c *client) getORASAuth() func(context.Context, string) (orasauth.Credential, error) {
	return func(ctx context.Context, registry string) (orasauth.Credential, error) {
		if c.auth == nil {
			return orasauth.EmptyCredential, nil
		}
		
		authObj, err := c.auth.Authenticator(registry)
		if err != nil {
			return orasauth.EmptyCredential, err
		}
		
		return auth.ToORASAuth(ctx, authObj, registry)
	}
}

// getGGCRAuth returns auth function for ggcr
func (c *client) getGGCRAuth() func() (authn.Authenticator, error) {
	return func() (authn.Authenticator, error) {
		if c.auth == nil {
			return authn.Anonymous, nil
		}
		
		// For ggcr, we need to extract registry from the current operation
		// This is a simplified approach; in production you might pass registry through
		authObj, err := c.auth.Authenticator("")
		if err != nil {
			return authn.Anonymous, err
		}
		
		return auth.ToGGCRAuth(authObj)
	}
}

// descriptorFromOCISpec converts OCI spec descriptor to our Descriptor
func (c *client) descriptorFromOCISpec(spec *ocispec.Descriptor, ref string) Descriptor {
	if spec == nil {
		return Descriptor{}
	}
	
	return Descriptor{
		Ref:         ref,
		Digest:      string(spec.Digest),
		Size:        spec.Size,
		MediaType:   spec.MediaType,
		Annotations: spec.Annotations,
		PushedAt:    time.Now(), // This would ideally come from registry
	}
}

// wrapError wraps an error with appropriate OCI context
func (c *client) wrapError(err error, operation, ref, registry string) error {
	if err == nil {
		return nil
	}
	
	// Check if it's already an OCIError
	var ociErr *OCIError
	if errors.As(err, &ociErr) {
		return err
	}
	
	// Determine error category from the error
	category := observability.GetErrorCategory(err)
	
	// Create new structured error
	switch category {
	case ErrorCategoryAuth:
		return observability.NewAuthError(operation, registry, err)
	case ErrorCategoryNetwork:
		return observability.NewNetworkError(operation, registry, err)
	case ErrorCategoryRegistry:
		httpStatus := observability.ExtractHTTPStatus(err)
		return observability.NewRegistryError(operation, registry, httpStatus, err)
	case ErrorCategoryValidation:
		return observability.NewValidationError(operation, ref, err)
	default:
		return observability.NewOCIError(err, category, operation).
			WithContext("reference", ref).
			WithContext("registry", registry)
	}
}

// recordMetrics records operation metrics if enabled
func (c *client) recordMetrics(operation, registry string, duration time.Duration, err error) {
	if !c.opts.EnableMetrics || c.opts.MetricsCallback == nil {
		return
	}
	
	// This would typically be stored in client state, but for now create new metrics
	metrics := observability.NewMetrics()
	metrics.RecordOperation(operation, registry, duration, err)
	
	// Call the user's metrics callback
	c.opts.MetricsCallback(metrics)
}

// recordFallbackAttempt records a fallback attempt in metrics
func (c *client) recordFallbackAttempt(artifactSuccess, imageSuccess bool) {
	if !c.opts.EnableMetrics || c.opts.MetricsCallback == nil {
		return
	}
	
	// This would typically be stored in client state, but for now create new metrics
	metrics := observability.NewMetrics()
	metrics.RecordFallback(artifactSuccess, imageSuccess)
	
	// Call the user's metrics callback
	c.opts.MetricsCallback(metrics)
}
package ociv2

import (
	"fmt"
	"io"
)

// LayerSpec defines a single layer to be pushed as part of an artifact
type LayerSpec struct {
	MediaType   string            // Required: media type of the layer
	Title       string            // Optional: sets OCI title annotation for the layer
	Annotations map[string]string // Optional: extra layer annotations
	Size        int64             // Required: size of the layer in bytes
	Reader      io.Reader         // Required: reader for the layer content
}

// PackOptions configures how to pack and push a multi-layer artifact
type PackOptions struct {
	// Artifact manifest fields
	ArtifactType        string            // Required for artifact manifests
	ManifestAnnotations map[string]string // OCI manifest annotations
	
	// Config blob (optional)
	Config          []byte // Optional config payload
	ConfigMediaType string // Media type for config (required if Config is set)
	
	// Layers (at least one required)
	Layers []LayerSpec // One or more blobs to push
	
	// Manifest type preferences
	PreferArtifactManifest bool // Try OCI 1.1 artifact manifest first (default: true)
	FallbackImageManifest  bool // Fallback to image manifest if needed (default: true)
}

// Validate checks that PackOptions has valid configuration
func (opts *PackOptions) Validate() error {
	// Must have at least one layer or config
	if len(opts.Layers) == 0 && len(opts.Config) == 0 {
		return fmt.Errorf("at least one layer or config is required")
	}
	
	// If config is provided, must have media type
	if len(opts.Config) > 0 && opts.ConfigMediaType == "" {
		return fmt.Errorf("config media type is required when config is provided")
	}
	
	// Validate each layer
	for i, layer := range opts.Layers {
		if layer.MediaType == "" {
			return fmt.Errorf("layer %d: media type is required", i)
		}
		if layer.Size < 0 {
			return fmt.Errorf("layer %d: size cannot be negative", i)
		}
		if layer.Reader == nil {
			return fmt.Errorf("layer %d: reader is required", i)
		}
	}
	
	// For artifact manifests, artifact type is required
	if opts.PreferArtifactManifest && opts.ArtifactType == "" {
		return fmt.Errorf("artifact type is required for artifact manifests")
	}
	
	return nil
}

// PulledLayer represents a layer pulled from an artifact with lazy loading
type PulledLayer struct {
	MediaType   string            // Media type of the layer
	Size        int64             // Size of the layer in bytes
	Digest      string            // Digest of the layer
	Annotations map[string]string // Layer annotations from manifest
	
	// Open lazily streams the blob content
	// Caller must close the returned ReadCloser
	Open func() (io.ReadCloser, error)
}

// PullResult contains the complete result of pulling an artifact
type PullResult struct {
	// Manifest metadata
	Descriptor  Descriptor        // Descriptor of the manifest itself
	ArtifactType string           // Artifact type (if artifact manifest)
	ManifestAnn map[string]string // Manifest-level annotations
	
	// Config blob (may be nil)
	Config          []byte // Config content (nil if no config)
	ConfigMediaType string // Config media type
	
	// Layers
	Layers []PulledLayer // All layers in the artifact
}

// GetLayer returns the layer at the specified index, or error if out of bounds
func (pr *PullResult) GetLayer(index int) (*PulledLayer, error) {
	if index < 0 || index >= len(pr.Layers) {
		return nil, fmt.Errorf("layer index %d out of bounds (have %d layers)", index, len(pr.Layers))
	}
	return &pr.Layers[index], nil
}

// GetLayerByMediaType returns the first layer matching the media type
func (pr *PullResult) GetLayerByMediaType(mediaType string) (*PulledLayer, error) {
	for _, layer := range pr.Layers {
		if layer.MediaType == mediaType {
			return &layer, nil
		}
	}
	return nil, fmt.Errorf("no layer found with media type %q", mediaType)
}

// HasConfig returns true if the artifact has a config blob
func (pr *PullResult) HasConfig() bool {
	return len(pr.Config) > 0
}

// LayerCount returns the number of layers in the artifact
func (pr *PullResult) LayerCount() int {
	return len(pr.Layers)
}
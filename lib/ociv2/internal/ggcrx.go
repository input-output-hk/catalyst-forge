package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// GGCRClient wraps go-containerregistry operations
type GGCRClient struct {
	plainHTTP bool
	userAgent string
	auth      func() (authn.Authenticator, error)
	client    *http.Client
}

// NewGGCRClient creates a new GGCR client
func NewGGCRClient(plainHTTP bool, userAgent string, authFunc func() (authn.Authenticator, error), httpClient *http.Client) *GGCRClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	
	return &GGCRClient{
		plainHTTP: plainHTTP,
		userAgent: userAgent,
		auth:      authFunc,
		client:    httpClient,
	}
}

// getRemoteOptions builds remote options for GGCR operations
func (c *GGCRClient) getRemoteOptions(ctx context.Context) []remote.Option {
	opts := []remote.Option{
		remote.WithContext(ctx),
	}
	
	// Add auth
	if c.auth != nil {
		if auth, err := c.auth(); err == nil && auth != nil {
			opts = append(opts, remote.WithAuth(auth))
		}
	}
	
	// Add transport with user agent
	if c.userAgent != "" {
		opts = append(opts, remote.WithUserAgent(c.userAgent))
	}
	
	// Add HTTP client
	if c.client != nil {
		opts = append(opts, remote.WithTransport(c.client.Transport))
	}
	
	// Handle plain HTTP
	if c.plainHTTP {
		opts = append(opts, remote.WithTransport(&plainHTTPTransport{
			base: c.client.Transport,
		}))
	}
	
	return opts
}

// PushJSONLayer pushes JSON as an image with empty config and JSON as layer
func (c *GGCRClient) PushJSONLayer(ctx context.Context, ref string, mediaType string, payload []byte, annotations map[string]string) (*ocispec.Descriptor, error) {
	// Parse reference
	r, err := name.ParseReference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference: %w", err)
	}
	
	// Create empty image
	img := empty.Image
	
	// Add JSON as a layer
	layer := static.NewLayer(payload, types.MediaType(mediaType))
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		return nil, fmt.Errorf("failed to add layer: %w", err)
	}
	
	// Set annotations on the image
	if len(annotations) > 0 {
		img = mutate.Annotations(img, annotations).(v1.Image)
	}
	
	// Push the image
	opts := c.getRemoteOptions(ctx)
	if err := remote.Write(r, img, opts...); err != nil {
		return nil, mapGGCRError(err)
	}
	
	// Get the digest
	d, err := img.Digest()
	if err != nil {
		return nil, fmt.Errorf("failed to get digest: %w", err)
	}
	
	// Get manifest to extract size
	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}
	
	return &ocispec.Descriptor{
		MediaType:   string(types.OCIManifestSchema1),
		Digest:      digest.Digest(d.String()),
		Size:        int64(len(manifestJSON)),
		Annotations: annotations,
	}, nil
}

// PushConfigAndLayer pushes config and tar layer as an image
func (c *GGCRClient) PushConfigAndLayer(ctx context.Context, ref string, cfg []byte, cfgMT string, tar io.Reader, size int64, layerMT string, annotations map[string]string) (*ocispec.Descriptor, error) {
	// Parse reference
	r, err := name.ParseReference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference: %w", err)
	}
	
	// Read tar data
	tarData, err := io.ReadAll(tar)
	if err != nil {
		return nil, fmt.Errorf("failed to read tar: %w", err)
	}
	
	// Create base image with custom config
	configFile := &v1.ConfigFile{
		Config: v1.Config{
			Labels: annotations,
		},
	}
	
	// Create image with config
	img, err := mutate.ConfigFile(empty.Image, configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to set config: %w", err)
	}
	
	// Add tar as layer
	layer := static.NewLayer(tarData, types.MediaType(layerMT))
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		return nil, fmt.Errorf("failed to add layer: %w", err)
	}
	
	// Set media types
	img = mutate.MediaType(img, types.OCIManifestSchema1)
	img = mutate.ConfigMediaType(img, types.MediaType(cfgMT))
	
	// Set annotations
	if len(annotations) > 0 {
		img = mutate.Annotations(img, annotations).(v1.Image)
	}
	
	// Push the image
	opts := c.getRemoteOptions(ctx)
	if err := remote.Write(r, img, opts...); err != nil {
		return nil, mapGGCRError(err)
	}
	
	// Get the digest
	d, err := img.Digest()
	if err != nil {
		return nil, fmt.Errorf("failed to get digest: %w", err)
	}
	
	// Get manifest size
	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}
	
	return &ocispec.Descriptor{
		MediaType:   string(types.OCIManifestSchema1),
		Digest:      digest.Digest(d.String()),
		Size:        int64(len(manifestJSON)),
		Annotations: annotations,
	}, nil
}

// PullJSONLayer pulls JSON from an image layer
func (c *GGCRClient) PullJSONLayer(ctx context.Context, ref string, wantMT string) ([]byte, *ocispec.Descriptor, error) {
	// Parse reference
	r, err := name.ParseReference(ref)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse reference: %w", err)
	}
	
	// Get the image
	opts := c.getRemoteOptions(ctx)
	img, err := remote.Image(r, opts...)
	if err != nil {
		return nil, nil, mapGGCRError(err)
	}
	
	// Get layers
	layers, err := img.Layers()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get layers: %w", err)
	}
	
	// Find the layer with matching media type or get the first layer
	var targetLayer v1.Layer
	for _, layer := range layers {
		mt, err := layer.MediaType()
		if err != nil {
			continue
		}
		
		if wantMT == "" || string(mt) == wantMT {
			targetLayer = layer
			break
		}
	}
	
	if targetLayer == nil && len(layers) > 0 {
		// Fallback to first layer if no match
		targetLayer = layers[0]
	}
	
	if targetLayer == nil {
		return nil, nil, fmt.Errorf("no layers found")
	}
	
	// Get layer content
	rc, err := targetLayer.Uncompressed()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get layer content: %w", err)
	}
	defer rc.Close()
	
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read layer: %w", err)
	}
	
	// Get manifest for descriptor
	d, err := img.Digest()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get digest: %w", err)
	}
	
	manifest, err := img.Manifest()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}
	
	// Get annotations
	annotations := make(map[string]string)
	if manifest.Annotations != nil {
		annotations = manifest.Annotations
	}
	
	return data, &ocispec.Descriptor{
		MediaType:   string(manifest.MediaType),
		Digest:      digest.Digest(d.String()),
		Size:        int64(len(manifestJSON)),
		Annotations: annotations,
	}, nil
}

// PullLayer pulls a specific layer from an image
func (c *GGCRClient) PullLayer(ctx context.Context, ref string, layerMT string) (io.ReadCloser, *ocispec.Descriptor, error) {
	// Parse reference
	r, err := name.ParseReference(ref)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse reference: %w", err)
	}
	
	// Get the image
	opts := c.getRemoteOptions(ctx)
	img, err := remote.Image(r, opts...)
	if err != nil {
		return nil, nil, mapGGCRError(err)
	}
	
	// Get layers
	layers, err := img.Layers()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get layers: %w", err)
	}
	
	// Find the layer with matching media type
	var targetLayer v1.Layer
	for _, layer := range layers {
		mt, err := layer.MediaType()
		if err != nil {
			continue
		}
		
		if string(mt) == layerMT {
			targetLayer = layer
			break
		}
	}
	
	if targetLayer == nil {
		return nil, nil, fmt.Errorf("layer with media type %s not found", layerMT)
	}
	
	// Get layer content
	rc, err := targetLayer.Compressed()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get layer content: %w", err)
	}
	
	// Get manifest for descriptor
	d, err := img.Digest()
	if err != nil {
		rc.Close()
		return nil, nil, fmt.Errorf("failed to get digest: %w", err)
	}
	
	manifest, err := img.Manifest()
	if err != nil {
		rc.Close()
		return nil, nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		rc.Close()
		return nil, nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}
	
	// Get annotations
	annotations := make(map[string]string)
	if manifest.Annotations != nil {
		annotations = manifest.Annotations
	}
	
	return rc, &ocispec.Descriptor{
		MediaType:   string(manifest.MediaType),
		Digest:      digest.Digest(d.String()),
		Size:        int64(len(manifestJSON)),
		Annotations: annotations,
	}, nil
}

// Head performs a HEAD request for a reference
func (c *GGCRClient) Head(ctx context.Context, ref string) (*ocispec.Descriptor, error) {
	// Parse reference
	r, err := name.ParseReference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference: %w", err)
	}
	
	// Get descriptor using HEAD
	opts := c.getRemoteOptions(ctx)
	desc, err := remote.Head(r, opts...)
	if err != nil {
		return nil, mapGGCRError(err)
	}
	
	return &ocispec.Descriptor{
		MediaType:   string(desc.MediaType),
		Digest:      digest.Digest(desc.Digest.String()),
		Size:        desc.Size,
		Annotations: desc.Annotations,
	}, nil
}

// Resolve fetches the manifest and returns a descriptor
func (c *GGCRClient) Resolve(ctx context.Context, ref string) (*ocispec.Descriptor, error) {
	// Parse reference
	r, err := name.ParseReference(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference: %w", err)
	}
	
	// Get the descriptor
	opts := c.getRemoteOptions(ctx)
	desc, err := remote.Get(r, opts...)
	if err != nil {
		return nil, mapGGCRError(err)
	}
	
	return &ocispec.Descriptor{
		MediaType:   string(desc.MediaType),
		Digest:      digest.Digest(desc.Digest.String()),
		Size:        desc.Size,
		Annotations: desc.Annotations,
	}, nil
}

// plainHTTPTransport allows plain HTTP connections
type plainHTTPTransport struct {
	base http.RoundTripper
}

func (t *plainHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Force HTTP
	if req.URL.Scheme == "https" {
		req.URL.Scheme = "http"
	}
	
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	
	return base.RoundTrip(req)
}

// mapGGCRError maps go-containerregistry errors to our standard errors
func mapGGCRError(err error) error {
	if err == nil {
		return nil
	}
	
	// Check for transport errors
	var transportErr *transport.Error
	if errors.As(err, &transportErr) {
		switch transportErr.StatusCode {
		case 401:
			return ErrUnauthorized
		case 403:
			return ErrForbidden
		case 404:
			return ErrNotFound
		case 408, 504:
			return ErrTimeout
		}
	}
	
	// Check error messages
	errStr := err.Error()
	if contains(errStr, "not found") {
		return ErrNotFound
	}
	if contains(errStr, "unauthorized") || contains(errStr, "401") {
		return ErrUnauthorized
	}
	if contains(errStr, "forbidden") || contains(errStr, "403") {
		return ErrForbidden
	}
	if contains(errStr, "timeout") {
		return ErrTimeout
	}
	if contains(errStr, "context canceled") {
		return ErrCanceled
	}
	if contains(errStr, "NAME_UNKNOWN") {
		return ErrNotFound
	}
	
	return err
}
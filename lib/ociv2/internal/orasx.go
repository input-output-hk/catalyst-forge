package internal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// ORASClient wraps ORAS operations
type ORASClient struct {
	plainHTTP bool
	userAgent string
	auth      func(context.Context, string) (auth.Credential, error)
}

// NewORASClient creates a new ORAS client
func NewORASClient(plainHTTP bool, userAgent string, authFunc func(context.Context, string) (auth.Credential, error)) *ORASClient {
	return &ORASClient{
		plainHTTP: plainHTTP,
		userAgent: userAgent,
		auth:      authFunc,
	}
}

// getRepository creates an ORAS repository for the given reference
func (c *ORASClient) getRepository(ctx context.Context, ref string, registryHost string) (*remote.Repository, error) {
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}
	
	// Configure client
	repo.PlainHTTP = c.plainHTTP
	
	// Set user agent
	if c.userAgent != "" {
		client := &auth.Client{
			Client: retry.DefaultClient,
		}
		client.SetUserAgent(c.userAgent)
		repo.Client = client
	}
	
	// Set auth
	if c.auth != nil {
		cred, err := c.auth(ctx, registryHost)
		if err == nil && (cred.Username != "" || cred.AccessToken != "" || cred.RefreshToken != "") {
			if repo.Client == nil {
				repo.Client = &auth.Client{
					Client: retry.DefaultClient,
					Credential: auth.StaticCredential(registryHost, cred),
				}
			} else if client, ok := repo.Client.(*auth.Client); ok {
				client.Credential = auth.StaticCredential(registryHost, cred)
			}
		}
	}
	
	return repo, nil
}

// PushConfigOnly pushes a JSON blob as an artifact config (no layers)
func (c *ORASClient) PushConfigOnly(ctx context.Context, ref string, registryHost string, mediaType string, payload []byte, annotations map[string]string) (*ocispec.Descriptor, error) {
	repo, err := c.getRepository(ctx, ref, registryHost)
	if err != nil {
		return nil, err
	}
	
	// Create memory store for the config
	memStore := memory.New()
	
	// Add config to store
	configDesc := ocispec.Descriptor{
		MediaType: mediaType,
		Size:      int64(len(payload)),
		Digest:    calcDigest(payload),
		Annotations: annotations,
	}
	
	if err := memStore.Push(ctx, configDesc, bytes.NewReader(payload)); err != nil {
		return nil, fmt.Errorf("failed to add config to store: %w", err)
	}
	
	// Pack and push
	manifestDesc, err := oras.PackManifest(ctx, memStore, oras.PackManifestVersion1_1, mediaType, oras.PackManifestOptions{
		ConfigDescriptor: &configDesc,
		ManifestAnnotations: annotations,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pack manifest: %w", err)
	}
	
	// Tag the manifest in the store before copying
	if err := memStore.Tag(ctx, manifestDesc, ref); err != nil {
		return nil, fmt.Errorf("failed to tag manifest: %w", err)
	}
	
	// Push to registry
	if _, err := oras.Copy(ctx, memStore, ref, repo, ref, oras.DefaultCopyOptions); err != nil {
		return nil, mapORASError(err)
	}
	
	return &manifestDesc, nil
}

// PushConfigAndLayer pushes config + tar layer as an artifact
func (c *ORASClient) PushConfigAndLayer(ctx context.Context, ref string, registryHost string, cfg []byte, cfgMT string, tar io.Reader, size int64, layerMT string, annotations map[string]string) (*ocispec.Descriptor, error) {
	repo, err := c.getRepository(ctx, ref, registryHost)
	if err != nil {
		return nil, err
	}
	
	// Create memory store
	memStore := memory.New()
	
	// Add config
	configDesc := ocispec.Descriptor{
		MediaType: cfgMT,
		Size:      int64(len(cfg)),
		Digest:    calcDigest(cfg),
	}
	
	if err := memStore.Push(ctx, configDesc, bytes.NewReader(cfg)); err != nil {
		return nil, fmt.Errorf("failed to add config: %w", err)
	}
	
	// Add layer
	layerData, err := io.ReadAll(tar)
	if err != nil {
		return nil, fmt.Errorf("failed to read tar: %w", err)
	}
	
	layerDesc := ocispec.Descriptor{
		MediaType: layerMT,
		Size:      int64(len(layerData)),
		Digest:    calcDigest(layerData),
	}
	
	if err := memStore.Push(ctx, layerDesc, bytes.NewReader(layerData)); err != nil {
		return nil, fmt.Errorf("failed to add layer: %w", err)
	}
	
	// Pack manifest with config and layer
	manifestDesc, err := oras.PackManifest(ctx, memStore, oras.PackManifestVersion1_1, "", oras.PackManifestOptions{
		ConfigDescriptor: &configDesc,
		Layers:          []ocispec.Descriptor{layerDesc},
		ManifestAnnotations: annotations,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pack manifest: %w", err)
	}
	
	// Tag the manifest in the store before copying
	if err := memStore.Tag(ctx, manifestDesc, ref); err != nil {
		return nil, fmt.Errorf("failed to tag manifest: %w", err)
	}
	
	// Push to registry
	if _, err := oras.Copy(ctx, memStore, ref, repo, ref, oras.DefaultCopyOptions); err != nil {
		return nil, mapORASError(err)
	}
	
	return &manifestDesc, nil
}

// PullConfig pulls an artifact config
func (c *ORASClient) PullConfig(ctx context.Context, ref string, registryHost string, wantMT string) ([]byte, *ocispec.Descriptor, error) {
	repo, err := c.getRepository(ctx, ref, registryHost)
	if err != nil {
		return nil, nil, err
	}
	
	// Get descriptor
	manifestDesc, err := repo.Resolve(ctx, ref)
	if err != nil {
		return nil, nil, mapORASError(err)
	}
	
	// Fetch manifest
	manifestData, err := content.FetchAll(ctx, repo, manifestDesc)
	if err != nil {
		return nil, nil, mapORASError(err)
	}
	
	// Parse manifest
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	
	// Check config media type
	if wantMT != "" && manifest.Config.MediaType != wantMT {
		return nil, nil, fmt.Errorf("unexpected media type: got %s, want %s", manifest.Config.MediaType, wantMT)
	}
	
	// Fetch config
	configData, err := content.FetchAll(ctx, repo, manifest.Config)
	if err != nil {
		return nil, nil, mapORASError(err)
	}
	
	desc := &ocispec.Descriptor{
		MediaType:   manifestDesc.MediaType,
		Digest:      manifestDesc.Digest,
		Size:        manifestDesc.Size,
		Annotations: manifest.Annotations,
	}
	
	return configData, desc, nil
}

// PullLayer pulls a specific layer from an artifact
func (c *ORASClient) PullLayer(ctx context.Context, ref string, registryHost string, layerMT string) (io.ReadCloser, *ocispec.Descriptor, error) {
	repo, err := c.getRepository(ctx, ref, registryHost)
	if err != nil {
		return nil, nil, err
	}
	
	// Get descriptor
	manifestDesc, err := repo.Resolve(ctx, ref)
	if err != nil {
		return nil, nil, mapORASError(err)
	}
	
	// Fetch manifest
	manifestData, err := content.FetchAll(ctx, repo, manifestDesc)
	if err != nil {
		return nil, nil, mapORASError(err)
	}
	
	// Parse manifest
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	
	// Find the layer with matching media type
	var targetLayer *ocispec.Descriptor
	for _, layer := range manifest.Layers {
		if layer.MediaType == layerMT {
			targetLayer = &layer
			break
		}
	}
	
	if targetLayer == nil {
		return nil, nil, fmt.Errorf("layer with media type %s not found", layerMT)
	}
	
	// Fetch layer
	rc, err := repo.Blobs().Fetch(ctx, *targetLayer)
	if err != nil {
		return nil, nil, mapORASError(err)
	}
	
	manifestDescResult := &ocispec.Descriptor{
		MediaType:   manifestDesc.MediaType,
		Digest:      manifestDesc.Digest,
		Size:        manifestDesc.Size,
		Annotations: manifest.Annotations,
	}
	
	return rc, manifestDescResult, nil
}

// Resolve resolves a reference to a descriptor
func (c *ORASClient) Resolve(ctx context.Context, ref string, registryHost string) (*ocispec.Descriptor, error) {
	repo, err := c.getRepository(ctx, ref, registryHost)
	if err != nil {
		return nil, err
	}
	
	desc, err := repo.Resolve(ctx, ref)
	if err != nil {
		return nil, mapORASError(err)
	}
	
	return &desc, nil
}

// Head performs a HEAD request for a reference
func (c *ORASClient) Head(ctx context.Context, ref string, registryHost string) (*ocispec.Descriptor, error) {
	// ORAS Resolve actually does a HEAD request for manifests
	return c.Resolve(ctx, ref, registryHost)
}

// Helper functions

// calcDigest calculates the digest of data
func calcDigest(data []byte) digest.Digest {
	return digest.Digest("sha256:" + sha256sum(data))
}

// sha256sum calculates SHA256 hash
func sha256sum(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// mapORASError maps ORAS errors to our standard errors
func mapORASError(err error) error {
	if err == nil {
		return nil
	}
	
	// Check for specific ORAS error types
	if errors.Is(err, errdef.ErrNotFound) {
		return ErrNotFound
	}
	
	// Check for HTTP errors in the error string
	errStr := err.Error()
	if contains(errStr, "401") || contains(errStr, "unauthorized") {
		return ErrUnauthorized
	}
	if contains(errStr, "403") || contains(errStr, "forbidden") {
		return ErrForbidden
	}
	if contains(errStr, "404") || contains(errStr, "not found") {
		return ErrNotFound
	}
	
	if contains(errStr, "timeout") {
		return ErrTimeout
	}
	if contains(errStr, "context canceled") {
		return ErrCanceled
	}
	
	return err
}


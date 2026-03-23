package oci

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure -pkg mocks -out mocks/client.go . Client

// Client is the interface for the OCI client
type Client interface {
	Pull(imageURL, destPath string) error
}

// OrasClient provides high-level OCI registry operations
type OrasClient struct {
	ctx      context.Context
	store    oras.Target
	destFS   fs.Filesystem // Filesystem for extraction destination
	verifier Verifier      // Optional signature verifier

	// Resource limits for DoS protection
	maxManifestSize     int64 // Maximum manifest size in bytes (default: 10MB)
	maxTotalExtractSize int64 // Maximum total extracted size in bytes (default: 1GB)
	maxFileSize         int64 // Maximum single file size in bytes (default: 100MB)
	maxFileCount        int   // Maximum number of files to extract (default: 10000)
}

// Option configures the OCI client
type Option func(*OrasClient) error

// WithContext sets a custom context for the client
func WithContext(ctx context.Context) Option {
	return func(c *OrasClient) error {
		c.ctx = ctx
		return nil
	}
}

// WithStore sets a custom ORAS store for the client
func WithStore(store oras.Target) Option {
	return func(c *OrasClient) error {
		if store == nil {
			return fmt.Errorf("store cannot be nil")
		}
		c.store = store
		return nil
	}
}

// WithFSStore creates a filesystem-based store using the provided filesystem
func WithFSStore(fs fs.Filesystem, storePath string) Option {
	return func(c *OrasClient) error {
		store, err := NewFilesystemStore(fs, storePath)
		if err != nil {
			return fmt.Errorf("failed to create filesystem store: %w", err)
		}
		c.store = store
		return nil
	}
}

// WithDestFS sets the filesystem for extraction destination
func WithDestFS(fs fs.Filesystem) Option {
	return func(c *OrasClient) error {
		if fs == nil {
			return fmt.Errorf("destination filesystem cannot be nil")
		}
		c.destFS = fs
		return nil
	}
}

// WithCosignVerification enables cosign signature verification with keyless OIDC
func WithCosignVerification(oidcIssuer, oidcSubject string) Option {
	return func(c *OrasClient) error {
		verifier, err := NewCosignVerifier(CosignOptions{
			OIDCIssuer:  oidcIssuer,
			OIDCSubject: oidcSubject,
		})
		if err != nil {
			return fmt.Errorf("failed to create cosign verifier: %w", err)
		}
		c.verifier = verifier
		return nil
	}
}

// WithCosignVerificationAdvanced enables cosign signature verification with advanced options
func WithCosignVerificationAdvanced(opts CosignOptions) Option {
	return func(c *OrasClient) error {
		verifier, err := NewCosignVerifier(opts)
		if err != nil {
			return fmt.Errorf("failed to create cosign verifier with advanced options: %w", err)
		}
		c.verifier = verifier
		return nil
	}
}

// WithCustomVerifier sets a custom signature verifier
func WithCustomVerifier(verifier Verifier) Option {
	return func(c *OrasClient) error {
		if verifier == nil {
			return fmt.Errorf("verifier cannot be nil")
		}
		c.verifier = verifier
		return nil
	}
}

// WithMaxManifestSize sets the maximum manifest size in bytes
func WithMaxManifestSize(size int64) Option {
	return func(c *OrasClient) error {
		if size <= 0 {
			return fmt.Errorf("max manifest size must be positive")
		}
		c.maxManifestSize = size
		return nil
	}
}

// WithMaxTotalExtractSize sets the maximum total extracted size in bytes
func WithMaxTotalExtractSize(size int64) Option {
	return func(c *OrasClient) error {
		if size <= 0 {
			return fmt.Errorf("max total extract size must be positive")
		}
		c.maxTotalExtractSize = size
		return nil
	}
}

// WithMaxFileSize sets the maximum single file size in bytes
func WithMaxFileSize(size int64) Option {
	return func(c *OrasClient) error {
		if size <= 0 {
			return fmt.Errorf("max file size must be positive")
		}
		c.maxFileSize = size
		return nil
	}
}

// WithMaxFileCount sets the maximum number of files to extract
func WithMaxFileCount(count int) Option {
	return func(c *OrasClient) error {
		if count <= 0 {
			return fmt.Errorf("max file count must be positive")
		}
		c.maxFileCount = count
		return nil
	}
}

// New creates a new OCI client with default options (in-memory store, OS filesystem for extraction)
func New(opts ...Option) (*OrasClient, error) {
	// Create default in-memory store for temporary storage
	defaultStore, err := NewFilesystemStore(billy.NewInMemoryFs(), "/tmp-oci-store")
	if err != nil {
		return nil, fmt.Errorf("failed to create default store: %w", err)
	}

	client := &OrasClient{
		ctx:                 context.Background(),
		store:               defaultStore,
		destFS:              billy.NewOsFs("/"), // Default to OS filesystem for extraction
		maxManifestSize:     10 * 1024 * 1024,   // 10MB
		maxTotalExtractSize: 1024 * 1024 * 1024, // 1GB
		maxFileSize:         100 * 1024 * 1024,  // 100MB
		maxFileCount:        10000,              // 10k files
	}

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	return client, nil
}

// Pull downloads an OCI image from a registry and extracts it to the specified directory
func (c *OrasClient) Pull(imageURL, destPath string) error {
	if imageURL == "" {
		return fmt.Errorf("image URL cannot be empty")
	}
	if destPath == "" {
		return fmt.Errorf("destination path cannot be empty")
	}
	if c.store == nil {
		return fmt.Errorf("no store configured - use WithStore() or WithFSStore() option")
	}

	repo, err := remote.NewRepository(imageURL)
	if err != nil {
		return fmt.Errorf("failed to create repository for %s: %w", imageURL, err)
	}

	// Configure Docker credential helpers
	storeOpts := credentials.StoreOptions{}
	credStore, err := credentials.NewStoreFromDocker(storeOpts)
	if err != nil {
		return fmt.Errorf("failed to create credential store: %w", err)
	}

	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore),
	}

	manifestDesc, err := oras.Copy(c.ctx, repo, imageURL, c.store, imageURL, oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("failed to pull OCI artifact %s: %w", imageURL, err)
	}

	// Verify signature if verifier is configured
	if c.verifier != nil {
		if err := c.verifier.Verify(c.ctx, imageURL, manifestDesc); err != nil {
			return fmt.Errorf("signature verification failed for %s: %w", imageURL, err)
		}
	}

	if err := c.extractArtifact(c.store, manifestDesc, destPath); err != nil {
		return fmt.Errorf("failed to extract artifact to %s: %w", destPath, err)
	}

	return nil
}

// extractArtifact reads the manifest and extracts tar layers to the destination
func (c *OrasClient) extractArtifact(store oras.Target, manifestDesc ocispec.Descriptor, destPath string) error {
	manifestReader, err := store.Fetch(c.ctx, manifestDesc)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer manifestReader.Close()

	// Limit manifest size to prevent memory exhaustion
	limitedReader := io.LimitReader(manifestReader, c.maxManifestSize)
	manifestBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Check if we hit the limit
	if int64(len(manifestBytes)) == c.maxManifestSize {
		// Try to read one more byte to see if manifest is larger than limit
		var buf [1]byte
		if n, _ := manifestReader.Read(buf[:]); n > 0 {
			return fmt.Errorf("manifest size exceeds limit of %d bytes", c.maxManifestSize)
		}
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	return c.extractTarLayers(store, manifest.Layers, destPath)
}

// extractTarLayers processes all layers and extracts tar content
func (c *OrasClient) extractTarLayers(store oras.Target, layers []ocispec.Descriptor, destPath string) error {
	var extractedLayers int
	var totalExtractedSize int64
	var totalFileCount int

	for _, layer := range layers {
		if c.isTarLayer(layer.MediaType) {
			extractedSize, fileCount, err := c.extractTarLayer(store, layer, destPath, totalExtractedSize, totalFileCount)
			if err != nil {
				return fmt.Errorf("failed to extract layer %s: %w", layer.Digest, err)
			}
			totalExtractedSize += extractedSize
			totalFileCount += fileCount
			extractedLayers++
		}
	}

	if extractedLayers == 0 {
		return fmt.Errorf("no tar layers found in OCI artifact")
	}

	return nil
}

// isTarLayer checks if a media type represents a tar layer
func (c *OrasClient) isTarLayer(mediaType string) bool {
	return mediaType == "application/vnd.oci.image.layer.v1.tar" ||
		mediaType == "application/vnd.oci.image.layer.v1.tar+gzip" ||
		mediaType == "application/octet-stream"
}

// extractTarLayer extracts a single tar layer to the destination using streaming
func (c *OrasClient) extractTarLayer(store oras.Target, layer ocispec.Descriptor, destPath string, currentTotalSize int64, currentFileCount int) (int64, int, error) {
	layerReader, err := store.Fetch(c.ctx, layer)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch layer: %w", err)
	}
	defer layerReader.Close()

	return c.streamExtractTar(layerReader, destPath, currentTotalSize, currentFileCount)
}

// streamExtractTar extracts tar data from a reader directly to filesystem with resource limits
func (c *OrasClient) streamExtractTar(reader io.Reader, destDir string, currentTotalSize int64, currentFileCount int) (int64, int, error) {
	if err := c.destFS.MkdirAll(destDir, 0755); err != nil {
		return 0, 0, fmt.Errorf("failed to create destination directory: %w", err)
	}

	tr := tar.NewReader(reader)
	var extractedSize int64
	var fileCount int

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return extractedSize, fileCount, fmt.Errorf("error reading tar: %w", err)
		}

		if header.Name == "." {
			continue
		}

		target := filepath.Join(destDir, header.Name)

		if !c.isSecurePath(target, destDir) {
			return extractedSize, fileCount, fmt.Errorf("invalid tar entry path (potential path traversal): %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := c.destFS.MkdirAll(target, 0755); err != nil {
				return extractedSize, fileCount, fmt.Errorf("failed to create directory %s: %w", target, err)
			}
			fileCount++

		case tar.TypeReg:
			// Check file size limit
			if header.Size > c.maxFileSize {
				return extractedSize, fileCount, fmt.Errorf("file %s size (%d bytes) exceeds max file size limit (%d bytes)",
					header.Name, header.Size, c.maxFileSize)
			}

			// Check total extraction size limit
			if currentTotalSize+extractedSize+header.Size > c.maxTotalExtractSize {
				return extractedSize, fileCount, fmt.Errorf("total extraction size would exceed limit of %d bytes",
					c.maxTotalExtractSize)
			}

			// Check file count limit
			if currentFileCount+fileCount+1 > c.maxFileCount {
				return extractedSize, fileCount, fmt.Errorf("file count would exceed limit of %d files", c.maxFileCount)
			}

			written, err := c.streamExtractFile(tr, target, header.Size)
			if err != nil {
				return extractedSize, fileCount, fmt.Errorf("failed to extract file %s: %w", target, err)
			}
			extractedSize += written
			fileCount++
		}
	}

	return extractedSize, fileCount, nil
}

// isSecurePath checks if the target path is within the destination directory
// Uses path.Clean and strings.HasPrefix instead of deprecated filepath.HasPrefix
func (c *OrasClient) isSecurePath(target, destDir string) bool {
	cleanDest := path.Clean(destDir)
	cleanTarget := path.Clean(target)

	cleanDest = filepath.ToSlash(cleanDest)
	cleanTarget = filepath.ToSlash(cleanTarget)

	return cleanTarget == cleanDest || strings.HasPrefix(cleanTarget, cleanDest+"/")
}

// streamExtractFile extracts a single file from tar reader using streaming with size limit enforcement
func (c *OrasClient) streamExtractFile(tr *tar.Reader, target string, expectedSize int64) (int64, error) {
	parentDir := filepath.Dir(target)
	if err := c.destFS.MkdirAll(parentDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create parent directory for %s: %w", target, err)
	}

	file, err := c.destFS.Create(target)
	if err != nil {
		return 0, fmt.Errorf("failed to create file %s: %w", target, err)
	}
	defer file.Close()

	// Use a LimitedReader to ensure we don't write more than expected
	limitedReader := io.LimitReader(tr, expectedSize)
	written, err := io.Copy(file, limitedReader)
	if err != nil {
		return written, fmt.Errorf("failed to write file %s: %w", target, err)
	}

	// Verify we wrote the expected amount
	if written != expectedSize {
		return written, fmt.Errorf("file %s size mismatch: expected %d bytes, wrote %d bytes", target, expectedSize, written)
	}

	return written, nil
}

package oci

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/errdef"
)

// FilesystemStore is a content store that uses our fs.Filesystem interface
// This allows us to work with any filesystem implementation (OS, in-memory, etc.)
type FilesystemStore struct {
	fs        fs.Filesystem
	root      string
	mu        sync.RWMutex
	manifests map[string]ocispec.Descriptor // tag -> descriptor
	blobs     map[string]bool               // digest -> exists
}

// NewFilesystemStore creates a new content store using the provided filesystem
func NewFilesystemStore(fs fs.Filesystem, root string) (*FilesystemStore, error) {
	// Ensure root directory exists
	if err := fs.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory %s: %w", root, err)
	}

	// Create blobs directory
	blobsDir := filepath.Join(root, "blobs", "sha256")
	if err := fs.MkdirAll(blobsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create blobs directory: %w", err)
	}

	return &FilesystemStore{
		fs:        fs,
		root:      root,
		manifests: make(map[string]ocispec.Descriptor),
		blobs:     make(map[string]bool),
	}, nil
}

// Close cleans up the store resources
func (s *FilesystemStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear in-memory maps
	s.manifests = nil
	s.blobs = nil
	return nil
}

// Fetch retrieves content by descriptor
func (s *FilesystemStore) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	path := s.blobPath(target.Digest.String())

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if blob exists
	exists, err := s.fs.Exists(path)
	if err != nil {
		return nil, fmt.Errorf("failed to check blob existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("blob %s: %w", target.Digest, errdef.ErrNotFound)
	}

	file, err := s.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open blob %s: %w", target.Digest, err)
	}

	return file, nil
}

// Push stores content with the expected descriptor
func (s *FilesystemStore) Push(ctx context.Context, expected ocispec.Descriptor, reader io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path := s.blobPath(expected.Digest.String())

	// Create the file
	file, err := s.fs.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create blob file %s: %w", expected.Digest, err)
	}
	defer file.Close()

	// Copy content and verify digest
	hasher := sha256.New()
	teeReader := io.TeeReader(reader, hasher)

	written, err := io.Copy(file, teeReader)
	if err != nil {
		return fmt.Errorf("failed to write blob %s: %w", expected.Digest, err)
	}

	// Verify size
	if expected.Size > 0 && written != expected.Size {
		return fmt.Errorf("size mismatch for blob %s: expected %d, got %d", expected.Digest, expected.Size, written)
	}

	// Verify digest
	actualDigest := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if actualDigest != expected.Digest.String() {
		return fmt.Errorf("digest mismatch for blob: expected %s, got %s", expected.Digest, actualDigest)
	}

	// Mark blob as existing
	s.mu.Lock()
	s.blobs[expected.Digest.String()] = true
	s.mu.Unlock()

	return nil
}

// Exists checks if content exists
func (s *FilesystemStore) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	path := s.blobPath(target.Digest.String())

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.fs.Exists(path)
}

// Resolve resolves a reference to a descriptor
func (s *FilesystemStore) Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error) {
	if err := ctx.Err(); err != nil {
		return ocispec.Descriptor{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if desc, ok := s.manifests[reference]; ok {
		return desc, nil
	}

	return ocispec.Descriptor{}, fmt.Errorf("reference %s: %w", reference, errdef.ErrNotFound)
}

// Tag adds a reference tag to a descriptor
func (s *FilesystemStore) Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.manifests[reference] = desc
	return nil
}

// Predecessors returns the nodes directly pointing to the current node
func (s *FilesystemStore) Predecessors(ctx context.Context, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// For simplicity, return empty slice
	// In a full implementation, you'd track relationships between descriptors
	return []ocispec.Descriptor{}, nil
}

// blobPath returns the filesystem path for a given digest
func (s *FilesystemStore) blobPath(digest string) string {
	// Remove algorithm prefix (e.g., "sha256:")
	if idx := strings.Index(digest, ":"); idx != -1 {
		digest = digest[idx+1:]
	}
	return filepath.Join(s.root, "blobs", "sha256", digest)
}

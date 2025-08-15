package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// HashStrings computes SHA256 hash of concatenated strings.
func HashStrings(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// HashBytes computes SHA256 hash of bytes.
func HashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// HashFile computes SHA256 hash of a file.
func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashReader computes SHA256 hash from a reader.
func HashReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("failed to hash reader: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputeIntentHash computes a deterministic intent hash for run caching.
func ComputeIntentHash(
	profile string,
	moduleDigest string,
	engineKind string,
	engineVersion string,
	kclVersion string,
	valuesHash string,
	ctxHash string,
) string {
	// Build a deterministic string representation
	parts := []string{
		"profile=" + profile,
		"module=" + moduleDigest,
		"engine=" + engineKind,
		"engineVer=" + engineVersion,
		"kclVer=" + kclVersion,
		"values=" + valuesHash,
		"ctx=" + ctxHash,
	}
	
	return HashStrings(strings.Join(parts, "&"))
}

// ValidateDigest validates a SHA256 digest string.
func ValidateDigest(digest string) error {
	// Remove sha256: prefix if present
	digest = strings.TrimPrefix(digest, "sha256:")
	
	// Check length (64 hex characters)
	if len(digest) != 64 {
		return fmt.Errorf("invalid digest length: expected 64, got %d", len(digest))
	}

	// Check if valid hex
	if _, err := hex.DecodeString(digest); err != nil {
		return fmt.Errorf("invalid hex in digest: %w", err)
	}

	return nil
}

// NormalizeDigest ensures digest has sha256: prefix.
func NormalizeDigest(digest string) string {
	if !strings.HasPrefix(digest, "sha256:") {
		return "sha256:" + digest
	}
	return digest
}

// StripDigestPrefix removes the sha256: prefix from a digest.
func StripDigestPrefix(digest string) string {
	return strings.TrimPrefix(digest, "sha256:")
}

// TruncateDigest returns a shortened version of the digest for display.
func TruncateDigest(digest string, length int) string {
	digest = StripDigestPrefix(digest)
	if len(digest) <= length {
		return digest
	}
	return digest[:length]
}

// HashWriter wraps a writer to compute hash while writing.
type HashWriter struct {
	w io.Writer
	h hash.Hash
}

// NewHashWriter creates a new hash writer.
func NewHashWriter(w io.Writer) *HashWriter {
	return &HashWriter{
		w: w,
		h: sha256.New(),
	}
}

// Write implements io.Writer.
func (hw *HashWriter) Write(p []byte) (n int, err error) {
	n, err = hw.w.Write(p)
	if err != nil {
		return n, err
	}
	hw.h.Write(p[:n])
	return n, nil
}

// Sum returns the current hash.
func (hw *HashWriter) Sum() string {
	return hex.EncodeToString(hw.h.Sum(nil))
}

// HashTeeReader creates a reader that hashes data as it's read.
func HashTeeReader(r io.Reader) (io.Reader, func() string) {
	h := sha256.New()
	tr := io.TeeReader(r, h)
	
	getHash := func() string {
		return hex.EncodeToString(h.Sum(nil))
	}
	
	return tr, getHash
}
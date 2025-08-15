package utils

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	// MaxManifestSize is the maximum size for a manifest (1MB)
	MaxManifestSize = 1024 * 1024
	
	// MaxConfigSize is the maximum size for a config blob (10MB)
	MaxConfigSize = 10 * 1024 * 1024
	
	// DefaultMaxBlobSize is the default maximum blob size (5GB)
	DefaultMaxBlobSize = 5 * 1024 * 1024 * 1024
)

var (
	// refPattern validates OCI reference format
	refPattern = regexp.MustCompile(`^(oci://)?([a-z0-9-_.]+(?::[0-9]+)?)/([a-z0-9-_./]+)(?:[:@]([a-z0-9-_.]+))?$`)
	
	// mediaTypePattern validates media type format
	mediaTypePattern = regexp.MustCompile(`^[a-z]+/[a-z0-9.+-]+$`)
	
	// digestPattern validates digest format
	digestPattern = regexp.MustCompile(`^[a-z0-9]+:[a-f0-9]+$`)
)

// ValidateReference validates an OCI reference format
func ValidateReference(ref string) error {
	if ref == "" {
		return fmt.Errorf("reference cannot be empty")
	}
	
	// Remove oci:// prefix for validation
	ref = strings.TrimPrefix(ref, "oci://")
	
	// Check for invalid characters
	if strings.ContainsAny(ref, " \t\n\r") {
		return fmt.Errorf("reference contains whitespace")
	}
	
	// Must have at least registry/repo format
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) < 2 {
		return fmt.Errorf("reference must include registry and repository")
	}
	
	// Validate registry part
	registry := parts[0]
	if registry == "" {
		return fmt.Errorf("registry cannot be empty")
	}
	
	// Check for localhost or domain format
	if !strings.Contains(registry, ".") && !strings.HasPrefix(registry, "localhost") {
		return fmt.Errorf("invalid registry format: %s", registry)
	}
	
	// Validate repository part
	repo := parts[1]
	if repo == "" {
		return fmt.Errorf("repository cannot be empty")
	}
	
	// Check for valid tag or digest
	if strings.Contains(repo, "@") {
		// Has digest
		parts := strings.Split(repo, "@")
		if len(parts) != 2 {
			return fmt.Errorf("invalid digest reference format")
		}
		if !digestPattern.MatchString(parts[1]) {
			return fmt.Errorf("invalid digest format: %s", parts[1])
		}
	}
	
	return nil
}

// ValidateMediaType validates a media type string
func ValidateMediaType(mediaType string) error {
	if mediaType == "" {
		return fmt.Errorf("media type cannot be empty")
	}
	
	if !mediaTypePattern.MatchString(mediaType) {
		return fmt.Errorf("invalid media type format: %s", mediaType)
	}
	
	// Check for known problematic media types
	if strings.HasPrefix(mediaType, "text/") && !strings.HasSuffix(mediaType, "+json") {
		return fmt.Errorf("text media types should use +json suffix for structured data")
	}
	
	return nil
}

// ValidateDigest validates a digest string
func ValidateDigest(digest string) error {
	if digest == "" {
		return fmt.Errorf("digest cannot be empty")
	}
	
	if !digestPattern.MatchString(digest) {
		return fmt.Errorf("invalid digest format: %s", digest)
	}
	
	// Check for supported algorithms
	if !strings.HasPrefix(digest, "sha256:") && !strings.HasPrefix(digest, "sha512:") {
		return fmt.Errorf("unsupported digest algorithm: %s", strings.Split(digest, ":")[0])
	}
	
	return nil
}

// ValidateBlobSize validates that a blob size is within limits
func ValidateBlobSize(size int64, maxSize int64) error {
	if size < 0 {
		return fmt.Errorf("blob size cannot be negative")
	}
	
	if maxSize == 0 {
		maxSize = DefaultMaxBlobSize
	}
	
	if size > maxSize {
		return fmt.Errorf("blob size %d exceeds maximum allowed size %d", size, maxSize)
	}
	
	return nil
}

// LimitedReader returns a reader that limits the amount of data read
func LimitedReader(r io.Reader, limit int64) io.Reader {
	return &limitedReader{
		r:     io.LimitReader(r, limit),
		limit: limit,
	}
}

type limitedReader struct {
	r     io.Reader
	limit int64
	read  int64
}

func (lr *limitedReader) Read(p []byte) (n int, err error) {
	n, err = lr.r.Read(p)
	lr.read += int64(n)
	
	if err == nil && lr.read >= lr.limit {
		// Check if there's more data
		peek := make([]byte, 1)
		if n, _ := lr.r.Read(peek); n > 0 {
			return n, fmt.Errorf("content exceeds maximum size of %d bytes", lr.limit)
		}
	}
	
	return n, err
}

// BufferedCopy copies from src to dst with a specified buffer size
func BufferedCopy(dst io.Writer, src io.Reader, bufferSize int) (int64, error) {
	if bufferSize <= 0 {
		bufferSize = 32 * 1024 // 32KB default
	}
	
	buf := make([]byte, bufferSize)
	return io.CopyBuffer(dst, src, buf)
}

// ValidateAnnotations validates annotation keys and values
func ValidateAnnotations(ann Annotations) error {
	for key, value := range ann {
		// Check key format (reverse domain notation recommended)
		if key == "" {
			return fmt.Errorf("annotation key cannot be empty")
		}
		
		// Check for excessively long keys
		if len(key) > 256 {
			return fmt.Errorf("annotation key %q exceeds maximum length of 256 characters", key)
		}
		
		// Check for excessively long values
		if len(value) > 4096 {
			return fmt.Errorf("annotation value for key %q exceeds maximum length of 4096 characters", key)
		}
		
		// Warn about non-ASCII characters
		for _, r := range key {
			if r > 127 {
				return fmt.Errorf("annotation key %q contains non-ASCII characters", key)
			}
		}
	}
	
	return nil
}
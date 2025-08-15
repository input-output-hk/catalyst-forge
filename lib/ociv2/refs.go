package ociv2

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/input-output-hk/catalyst-forge/lib/ociv2/observability"
)

// IsDigestRef checks if a reference contains a digest (@sha256:...)
func IsDigestRef(ref string) bool {
	return strings.Contains(ref, "@sha256:") || strings.Contains(ref, "@sha512:")
}

// ensureDigest ensures the Descriptor has a canonical reference with digest
func ensureDigest(ref string, d Descriptor) Descriptor {
	if IsDigestRef(ref) {
		// Already has digest, use as-is
		d.Ref = ref
	} else if d.Digest != "" {
		// Add digest to ref
		baseRef := strings.TrimSuffix(ref, ":")
		// Remove any tag if present
		if idx := strings.LastIndex(baseRef, ":"); idx > 0 {
			// Check if this is a tag (not a port)
			afterColon := baseRef[idx+1:]
			if !strings.Contains(afterColon, "/") {
				baseRef = baseRef[:idx]
			}
		}
		d.Ref = fmt.Sprintf("%s@%s", baseRef, d.Digest)
	} else {
		// No digest available, keep ref as-is
		d.Ref = ref
	}
	return d
}

// parseRef parses and validates an OCI reference
func parseRef(ref string) (name.Reference, error) {
	// Handle oci:// scheme
	ref = NormalizeRef(ref)
	
	// Try to parse as a digest reference first
	if IsDigestRef(ref) {
		return name.ParseReference(ref, name.WeakValidation)
	}
	
	// Parse as tag reference
	return name.ParseReference(ref, name.WeakValidation)
}

// NormalizeRef normalizes a reference by handling oci:// scheme and ensuring proper format
func NormalizeRef(ref string) string {
	// Remove oci:// prefix if present
	ref = strings.TrimPrefix(ref, "oci://")
	
	// Ensure we have a valid reference format
	// If no tag or digest, default to :latest
	if !strings.Contains(ref, "@") && !strings.Contains(ref, ":") {
		ref = ref + ":latest"
	} else if strings.Count(ref, ":") == 1 {
		// Check if the colon is part of a port number (e.g., localhost:5000/image)
		parts := strings.Split(ref, "/")
		if len(parts) > 1 {
			// If we have multiple parts, check if the first part looks like host:port
			firstPart := parts[0]
			if strings.Contains(firstPart, ":") {
				// This looks like host:port, add :latest
				ref = ref + ":latest"
			}
		}
		// If we have only one part (like "nginx:latest"), it already has a tag, don't add :latest
	}
	
	return ref
}

// toCanonical converts a reference to its canonical form (with digest if available)
func toCanonical(ref string, digest string) string {
	ref = NormalizeRef(ref)
	
	if digest == "" {
		return ref
	}
	
	// Remove any existing tag or digest
	if idx := strings.LastIndex(ref, "@"); idx > 0 {
		ref = ref[:idx]
	} else if idx := strings.LastIndex(ref, ":"); idx > 0 {
		// Check if this is a tag (not a port)
		afterColon := ref[idx+1:]
		if !strings.Contains(afterColon, "/") {
			ref = ref[:idx]
		}
	}
	
	return fmt.Sprintf("%s@%s", ref, digest)
}

// extractRegistry extracts the registry hostname from a reference
func extractRegistry(ref string) (string, error) {
	ref = NormalizeRef(ref)
	
	parsed, err := parseRef(ref)
	if err != nil {
		return "", fmt.Errorf("failed to parse reference: %w", err)
	}
	
	return parsed.Context().RegistryStr(), nil
}

// validateRef validates an OCI reference
func validateRef(ref string) error {
	// Check for insecure schemes
	if strings.HasPrefix(ref, "http://") {
		return observability.ErrInsecureRef
	}
	
	// Remove oci:// for parsing
	ref = NormalizeRef(ref)
	
	// Try to parse the reference
	_, err := parseRef(ref)
	if err != nil {
		return fmt.Errorf("%w: %s", observability.ErrInvalidRef, err.Error())
	}
	
	return nil
}

// isLocalRegistry checks if a reference points to a local registry
func isLocalRegistry(ref string) bool {
	ref = NormalizeRef(ref)
	
	// Common local registry patterns
	localPatterns := []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"host.docker.internal",
	}
	
	for _, pattern := range localPatterns {
		if strings.HasPrefix(ref, pattern+":") || strings.HasPrefix(ref, pattern+"/") {
			return true
		}
	}
	
	// Check if it's a local IP
	if parts := strings.Split(ref, "/"); len(parts) > 0 {
		host := parts[0]
		if colonIdx := strings.Index(host, ":"); colonIdx > 0 {
			host = host[:colonIdx]
		}
		
		// Check for private IP ranges
		if strings.HasPrefix(host, "10.") ||
			strings.HasPrefix(host, "172.") ||
			strings.HasPrefix(host, "192.168.") {
			return true
		}
	}
	
	return false
}

// splitRefParts splits a reference into registry, repository, and tag/digest parts
func splitRefParts(ref string) (registry, repository, tagOrDigest string, err error) {
	ref = NormalizeRef(ref)
	
	parsed, err := parseRef(ref)
	if err != nil {
		return "", "", "", err
	}
	
	registry = parsed.Context().RegistryStr()
	repository = parsed.Context().RepositoryStr()
	
	// Extract tag or digest
	if tagged, ok := parsed.(name.Tag); ok {
		tagOrDigest = tagged.TagStr()
	} else if digested, ok := parsed.(name.Digest); ok {
		tagOrDigest = digested.DigestStr()
	}
	
	return registry, repository, tagOrDigest, nil
}

// makeOCIURL converts a reference back to oci:// format
func makeOCIURL(ref string) string {
	ref = NormalizeRef(ref)
	if !strings.HasPrefix(ref, "oci://") {
		return "oci://" + ref
	}
	return ref
}

// isHTTPSRegistry checks if we should use HTTPS for a registry
func isHTTPSRegistry(ref string, plainHTTP bool) bool {
	// If plainHTTP is explicitly set, honor it
	if plainHTTP {
		return false
	}
	
	// Local registries can use HTTP
	if isLocalRegistry(ref) {
		return false
	}
	
	// Default to HTTPS for remote registries
	return true
}

// parseRegistryURL parses a registry URL and returns the base URL
func parseRegistryURL(registry string, useHTTPS bool) (*url.URL, error) {
	scheme := "https"
	if !useHTTPS {
		scheme = "http"
	}
	
	// Add scheme if not present
	if !strings.HasPrefix(registry, "http://") && !strings.HasPrefix(registry, "https://") {
		registry = scheme + "://" + registry
	}
	
	u, err := url.Parse(registry)
	if err != nil {
		return nil, fmt.Errorf("invalid registry URL: %w", err)
	}
	
	return u, nil
}
package internal

import (
	"strings"
	"time"
)

// GHCRSpecificOptions contains GHCR-specific configuration
type GHCRSpecificOptions struct {
	// PreferArtifacts indicates GHCR has excellent artifact support
	PreferArtifacts bool
	// OptimalTimeout for GHCR operations
	OptimalTimeout time.Duration
	// RequiresAuthentication indicates if auth is typically required
	RequiresAuthentication bool
}

// IsGHCRRegistry checks if the registry is GitHub Container Registry
func IsGHCRRegistry(registry string) bool {
	return strings.ToLower(registry) == "ghcr.io"
}

// OptimizeForGHCR returns GHCR-optimized settings
func OptimizeForGHCR() GHCRSpecificOptions {
	return GHCRSpecificOptions{
		PreferArtifacts:        true,  // GHCR has excellent OCI artifact support
		OptimalTimeout:         3 * time.Minute, // GHCR is generally fast
		RequiresAuthentication: true,  // GHCR typically requires auth even for public repos
	}
}

// IsGHCRManifestError checks if an error is specifically related to GHCR manifest issues
func IsGHCRManifestError(err error) bool {
	if err == nil {
		return false
	}
	
	errMsg := strings.ToLower(err.Error())
	
	// GHCR-specific error patterns (rare since GHCR supports artifacts well)
	ghcrPatterns := []string{
		"package does not exist",
		"version does not exist", 
		"insufficient permissions",
		"authentication required",
		"forbidden",
		"rate limit exceeded",
	}
	
	for _, pattern := range ghcrPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	
	return false
}

// GHCRAuthHelper provides GHCR-specific authentication assistance
type GHCRAuthHelper struct {
	Token     string
	Username  string
	Namespace string
}

// NewGHCRAuthHelper creates a new GHCR auth helper
func NewGHCRAuthHelper(token, username string) *GHCRAuthHelper {
	return &GHCRAuthHelper{
		Token:    token,
		Username: username,
	}
}

// ExtractNamespace extracts the namespace from a GHCR reference
func (h *GHCRAuthHelper) ExtractNamespace(ref string) string {
	// GHCR format: ghcr.io/{namespace}/{package}:{tag}
	parts := strings.Split(ref, "/")
	if len(parts) >= 2 && parts[0] == "ghcr.io" {
		return parts[1]
	}
	return ""
}

// IsPublicNamespace checks if the namespace typically allows anonymous access
func (h *GHCRAuthHelper) IsPublicNamespace(namespace string) bool {
	// Some well-known public namespaces on GHCR
	publicNamespaces := map[string]bool{
		"library":            true,
		"docker":             true,
		"microsoft":          true,
		"github":             true,
		"actions":            true,
		"homebrew":           true,
	}
	
	return publicNamespaces[strings.ToLower(namespace)]
}

// ShouldRetryWithAuth determines if a 401/403 error should trigger auth retry
func (h *GHCRAuthHelper) ShouldRetryWithAuth(err error, namespace string) bool {
	if err == nil {
		return false
	}
	
	errMsg := strings.ToLower(err.Error())
	
	// Check for auth-related errors
	authErrors := []string{
		"unauthorized",
		"authentication required", 
		"forbidden",
		"insufficient permissions",
		"401",
		"403",
	}
	
	for _, authErr := range authErrors {
		if strings.Contains(errMsg, authErr) {
			// If we have a token and this isn't a known public namespace, retry with auth
			return h.Token != "" && !h.IsPublicNamespace(namespace)
		}
	}
	
	return false
}

// GetOptimalConcurrency returns optimal concurrent operation settings for GHCR
func GetOptimalConcurrency() int {
	// GHCR can handle moderate concurrency well
	return 5
}

// ShouldUseGitHubToken determines if GitHub token should be used for auth
func ShouldUseGitHubToken(ref string) bool {
	// Always use GitHub token for GHCR if available
	return IsGHCRRegistry(extractRegistryFromRef(ref))
}

// extractRegistryFromRef is a helper to extract registry from full reference
func extractRegistryFromRef(ref string) string {
	// Simple extraction - in production this would use the same logic as extractRegistry
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
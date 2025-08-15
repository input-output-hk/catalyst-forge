package internal

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// RegistryCapabilities tracks what a registry supports
type RegistryCapabilities struct {
	SupportsArtifactManifest bool
	SupportsImageManifest    bool
	RegistryType             RegistryType
	ErrorSeen                error
}

// RegistryType identifies specific registry implementations
type RegistryType int

const (
	RegistryTypeUnknown RegistryType = iota
	RegistryTypeDockerHub
	RegistryTypeECR
	RegistryTypeGHCR
	RegistryTypeGCR
	RegistryTypeACR
	RegistryTypeQuay
	RegistryTypeGeneric
)

// String returns the string representation of RegistryType
func (r RegistryType) String() string {
	switch r {
	case RegistryTypeDockerHub:
		return "docker.io"
	case RegistryTypeECR:
		return "ecr"
	case RegistryTypeGHCR:
		return "ghcr.io"
	case RegistryTypeGCR:
		return "gcr.io"
	case RegistryTypeACR:
		return "azurecr.io"
	case RegistryTypeQuay:
		return "quay.io"
	case RegistryTypeGeneric:
		return "generic"
	default:
		return "unknown"
	}
}

// DetectRegistryType identifies the registry type from hostname
func DetectRegistryType(hostname string) RegistryType {
	hostname = strings.ToLower(hostname)
	
	switch {
	case hostname == "docker.io" || hostname == "registry-1.docker.io":
		return RegistryTypeDockerHub
	case strings.HasSuffix(hostname, ".amazonaws.com") && strings.Contains(hostname, "ecr"):
		return RegistryTypeECR
	case hostname == "ghcr.io":
		return RegistryTypeGHCR
	case strings.HasSuffix(hostname, "gcr.io"):
		return RegistryTypeGCR
	case strings.HasSuffix(hostname, "azurecr.io"):
		return RegistryTypeACR
	case hostname == "quay.io":
		return RegistryTypeQuay
	default:
		return RegistryTypeGeneric
	}
}

// ShouldFallbackToImageManifest determines if an error indicates we should retry with image manifest
func ShouldFallbackToImageManifest(err error, registryType RegistryType) bool {
	if err == nil {
		return false
	}
	
	// Check for HTTP status codes that indicate artifact manifest rejection
	if httpErr := extractHTTPError(err); httpErr != nil {
		switch httpErr.StatusCode {
		case http.StatusBadRequest, // 400 - Bad Request (malformed artifact manifest)
			 http.StatusUnsupportedMediaType, // 415 - Unsupported Media Type
			 http.StatusNotImplemented, // 501 - Not Implemented
			 http.StatusBadGateway: // 502 - Bad Gateway (some proxies reject unknown content types)
			return true
		}
	}
	
	// Check for ORAS-specific errors that indicate unsupported manifest types
	// Note: Using string matching instead of errdef functions for broader compatibility
	
	// Registry-specific patterns
	errMsg := strings.ToLower(err.Error())
	
	// Common patterns that indicate artifact manifest rejection
	rejectionPatterns := []string{
		"unsupported manifest type",
		"unsupported media type",
		"unknown manifest schema",
		"invalid manifest",
		"artifact manifest not supported",
		"oci artifact manifest not supported",
		"application/vnd.oci.artifact.manifest.v1+json",
	}
	
	for _, pattern := range rejectionPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	
	// Registry-specific error patterns
	switch registryType {
	case RegistryTypeECR:
		// ECR has specific error messages for unsupported manifests
		ecrPatterns := []string{
			"manifest blob unknown",
			"unsupported manifest media type",
			"invalid image manifest",
		}
		for _, pattern := range ecrPatterns {
			if strings.Contains(errMsg, pattern) {
				return true
			}
		}
		
	case RegistryTypeDockerHub:
		// Docker Hub sometimes returns cryptic errors for artifact manifests
		dockerPatterns := []string{
			"invalid json",
			"unknown blob",
			"manifest invalid",
		}
		for _, pattern := range dockerPatterns {
			if strings.Contains(errMsg, pattern) {
				return true
			}
		}
	}
	
	return false
}

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Message    string
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// NewHTTPError creates a new HTTPError
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// extractHTTPError attempts to extract HTTP status code from various error types
func extractHTTPError(err error) *HTTPError {
	if err == nil {
		return nil
	}
	
	// Check if it's already an HTTPError
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr
	}
	
	// Try to extract from common error patterns (using string matching for broad compatibility)
	errMsg := err.Error()
	errLower := strings.ToLower(errMsg)
	if strings.Contains(errLower, "not found") || strings.Contains(errLower, "404") {
		return &HTTPError{StatusCode: http.StatusNotFound, Message: "not found"}
	}
	if strings.Contains(errLower, "unauthorized") || strings.Contains(errLower, "401") {
		return &HTTPError{StatusCode: http.StatusUnauthorized, Message: "unauthorized"}
	}
	if strings.Contains(errLower, "forbidden") || strings.Contains(errLower, "403") {
		return &HTTPError{StatusCode: http.StatusForbidden, Message: "forbidden"}
	}
	if strings.Contains(errLower, "unsupported") || strings.Contains(errLower, "415") {
		return &HTTPError{StatusCode: http.StatusUnsupportedMediaType, Message: "unsupported"}
	}
	
	// Try to parse other common HTTP error patterns from error message
	
	// Look for "HTTP 4xx" or "status code 4xx" patterns
	if strings.Contains(errMsg, "400") || strings.Contains(errMsg, "bad request") {
		return &HTTPError{StatusCode: http.StatusBadRequest, Message: "bad request"}
	}
	if strings.Contains(errMsg, "415") || strings.Contains(errMsg, "unsupported media type") {
		return &HTTPError{StatusCode: http.StatusUnsupportedMediaType, Message: "unsupported media type"}
	}
	if strings.Contains(errMsg, "501") || strings.Contains(errMsg, "not implemented") {
		return &HTTPError{StatusCode: http.StatusNotImplemented, Message: "not implemented"}
	}
	
	return nil
}

// GetRegistrySpecificOptions returns optimal settings for different registry types
func GetRegistrySpecificOptions(registryType RegistryType) (preferArtifact, fallbackImage bool) {
	switch registryType {
	case RegistryTypeGHCR:
		// GHCR has excellent artifact manifest support
		return true, true
		
	case RegistryTypeGCR:
		// GCR supports artifacts well
		return true, true
		
	case RegistryTypeACR:
		// Azure Container Registry supports artifacts
		return true, true
		
	case RegistryTypeQuay:
		// Quay.io supports artifacts
		return true, true
		
	case RegistryTypeECR:
		// ECR support varies by region/version, safer to try artifact first but fallback
		return true, true
		
	case RegistryTypeDockerHub:
		// Docker Hub has limited artifact support, prefer image manifests
		return false, true
		
	default:
		// For unknown registries, try artifact first with fallback
		return true, true
	}
}

// LogRegistryCompatibility logs registry compatibility information
func LogRegistryCompatibility(logger func(msg string, kv ...any), registryType RegistryType, caps RegistryCapabilities) {
	if logger == nil {
		return
	}
	
	logger("oci.registry.compat", 
		"type", registryType.String(),
		"artifact_support", caps.SupportsArtifactManifest,
		"image_support", caps.SupportsImageManifest,
		"error", caps.ErrorSeen,
	)
}
package ociv2

import (
	"context"
	"time"
)

// Descriptor represents an OCI artifact with its metadata
type Descriptor struct {
	Ref         string            // canonical ref with @sha256:...
	Digest      string            // sha256:...
	Size        int64             // size in bytes
	MediaType   string            // MIME type of the content
	Annotations map[string]string // OCI annotations
	PushedAt    time.Time         // timestamp when pushed
}

// Well-known media types for Forge artifacts
const (
	// Forge-specific media types
	MTReleaseConfig = "application/vnd.forge.release+json"
	MTRenderedIndex = "application/vnd.forge.rendered.index.v1+json"
	MTRenderedTarGz = "application/vnd.forge.rendered.layer.v1.tar+gzip"
	
	// OCI standard media types
	MTOCIEmptyJSON     = "application/vnd.oci.empty.v1+json"
	MTOCIImageManifest = "application/vnd.oci.image.manifest.v1+json"
	MTOCIImageIndex    = "application/vnd.oci.image.index.v1+json"
	MTOCIArtifactManifest = "application/vnd.oci.artifact.manifest.v1+json"
)

// Error types are now in the observability package
// Use the error variables from observability package directly

// Annotations is a helper type for OCI annotations
type Annotations map[string]string

// Signing types are now in the signing package
// Use the type aliases from compat.go for backward compatibility

// JSONSelector extracts a JSON document from a pulled artifact
type JSONSelector interface {
	// Extract a JSON document (bytes) from a pulled artifact
	Extract(ctx context.Context, pr *PullResult) ([]byte, error)
}

// JSONValidator validates JSON documents
type JSONValidator interface {
	// Validate the given JSON bytes. Returns nil if valid
	Validate(ctx context.Context, doc []byte) error
}

// JSONValidation combines a selector and validator with a name for reporting
type JSONValidation struct {
	Name      string        // for reporting
	Selector  JSONSelector  // where the JSON comes from (config, layer, annotations, etc.)
	Validator JSONValidator // how to validate it (e.g., CUE or JSON Schema)
}

// ShapeValidationSpec defines shape constraints for an artifact
type ShapeValidationSpec struct {
	// Manifest-level constraints
	RequireArtifactType    string   // Required artifact type (exact match)
	RequireConfigMediaType string   // Required config media type (exact match)
	AllowedLayerMediaTypes []string // Allowed layer media types (if empty, any allowed)
	
	// Layer count constraints
	RequireLayerCount *struct {
		Min int
		Max int
	}
	
	// Required manifest annotations
	RequireManifestAnn map[string]bool // Keys that must be present
}

// VerifyOptions configures artifact verification
type VerifyOptions struct {
	// Signature verification (already in your package)
	RequireSignature bool
	
	// Shape checks
	Shape *ShapeValidationSpec
	
	// Optional JSON document validations to run after pull+signature+shape
	JSONValidations []JSONValidation
}

// ValidationReport contains the results of artifact verification
type ValidationReport struct {
	// Overall result
	Valid bool
	
	// Individual check results
	SignatureValid bool
	ShapeValid     bool
	JSONValid      bool
	
	// Error details
	SignatureError error
	ShapeError     error
	JSONErrors     map[string]error // Keyed by JSONValidation.Name
}
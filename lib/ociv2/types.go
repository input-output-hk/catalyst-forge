package ociv2

import (
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
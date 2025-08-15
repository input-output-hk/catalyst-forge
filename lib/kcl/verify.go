package kcl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Verify verifies a KCL module in the registry.
func Verify(ctx context.Context, ociCli OCI, ref ModuleRef, opts VerifyOptions) (string, []byte, error) {
	// Build reference string
	refStr := buildReference(ref)

	// Perform verification
	report, err := ociCli.VerifyArtifact(refStr, opts.RequireSignature)
	if err != nil {
		return "", nil, fmt.Errorf("failed to verify artifact: %w", err)
	}

	// Check signature if required
	if opts.RequireSignature && !report.SignatureValid {
		return "", nil, ErrSignatureInvalid
	}

	// Validate shape based on profile
	if err := validateShape(report, opts.Profile); err != nil {
		return "", nil, fmt.Errorf("shape validation failed: %w", err)
	}

	// Extract or synthesize metadata
	meta, err := extractMetadata(report, opts.Profile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// TODO: Add CUE schema validation when lib/ociv2/validate/cue is available

	return report.Digest, meta, nil
}

// buildReference builds a reference string from ModuleRef.
func buildReference(ref ModuleRef) string {
	if ref.Dig != "" {
		// Prefer digest
		repo := strings.TrimPrefix(ref.Repo, "oci://")
		return fmt.Sprintf("%s@%s", repo, normalizeDigest(ref.Dig))
	}

	if ref.Tag != "" {
		repo := strings.TrimPrefix(ref.Repo, "oci://")
		return fmt.Sprintf("%s:%s", repo, ref.Tag)
	}

	// Default to latest
	repo := strings.TrimPrefix(ref.Repo, "oci://")
	return fmt.Sprintf("%s:latest", repo)
}

// normalizeDigest ensures digest has sha256: prefix.
func normalizeDigest(digest string) string {
	if !strings.HasPrefix(digest, "sha256:") {
		return "sha256:" + digest
	}
	return digest
}

// validateShape validates the artifact shape based on profile.
func validateShape(report *VerificationReport, profile Profile) error {
	if !report.ShapeValid {
		return ErrShapeMismatch
	}

	// Additional profile-specific validation would go here
	// This would check layer types, counts, etc. based on the profile

	return nil
}

// extractMetadata extracts metadata from the verification report.
func extractMetadata(report *VerificationReport, profile Profile) ([]byte, error) {
	switch profile {
	case ProfileCompat:
		return extractCompatMetadata(report)
	case ProfileStrict:
		return extractStrictMetadata(report)
	default:
		return nil, fmt.Errorf("unsupported profile: %s", profile)
	}
}

// extractCompatMetadata synthesizes metadata from annotations for compat profile.
func extractCompatMetadata(report *VerificationReport) ([]byte, error) {
	// In compat mode, metadata is derived from manifest annotations
	meta := ModuleMeta{
		Name:        getAnnotation(report, "io.kcl.name", "org.opencontainers.image.title"),
		Version:     getAnnotation(report, "io.kcl.version", "org.opencontainers.image.version"),
		Description: getAnnotation(report, "io.kcl.description", "org.opencontainers.image.description"),
		Sum:         getAnnotation(report, "io.kcl.sum", ""),
	}

	// Parse authors from annotation
	authorsStr := getAnnotation(report, "org.opencontainers.image.authors", "")
	if authorsStr != "" {
		meta.Authors = strings.Split(authorsStr, ", ")
	}

	meta.License = getAnnotation(report, "org.opencontainers.image.licenses", "")
	meta.Repository = getAnnotation(report, "org.opencontainers.image.source", "")
	meta.Homepage = getAnnotation(report, "org.opencontainers.image.url", "")

	// Add all annotations
	if annotations, ok := report.Details["annotations"].(map[string]string); ok {
		meta.Annotations = annotations
	}

	return json.Marshal(meta)
}

// extractStrictMetadata extracts metadata from meta.json layer for strict profile.
func extractStrictMetadata(report *VerificationReport) ([]byte, error) {
	// In strict mode, metadata comes from the meta.json layer
	// Look for the meta.json layer content in Details

	if report.Details == nil {
		return nil, fmt.Errorf("no artifact details available")
	}

	// First check if metaLayer is directly available
	if metaLayer, ok := report.Details["metaLayer"].([]byte); ok {
		// Validate the metadata against strict schema
		var meta ModuleMeta
		if err := json.Unmarshal(metaLayer, &meta); err != nil {
			return nil, fmt.Errorf("invalid metadata JSON: %w", err)
		}

		// Ensure required fields for strict profile
		if meta.Name == "" || meta.Version == "" || meta.Sum == "" {
			return nil, fmt.Errorf("strict profile requires name, version, and sum in metadata")
		}

		// Add profile indicator
		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string)
		}
		meta.Annotations["dev.catalyst.forge.profile"] = "strict"

		return json.Marshal(meta)
	}

	// Check if we have manifest information to identify the meta layer
	if manifest, ok := report.Details["manifest"].(map[string]interface{}); ok {
		if layers, ok := manifest["layers"].([]interface{}); ok && len(layers) >= 2 {
			// Strict profile should have exactly 2 layers
			for _, layer := range layers {
				if layerMap, ok := layer.(map[string]interface{}); ok {
					mediaType, _ := layerMap["mediaType"].(string)
					if mediaType == "application/vnd.projectcatalyst.kcl.module.meta.v1+json" {
						// Try to retrieve content from report details if present
						if lm, ok := report.Details["layers"].(map[string][]byte); ok {
							digestStr, _ := layerMap["digest"].(string)
							if digestStr != "" {
								if b, exists := lm[digestStr]; exists {
									return b, nil
								}
								if strings.HasPrefix(digestStr, "sha256:") {
									if b, exists := lm[strings.TrimPrefix(digestStr, "sha256:")]; exists {
										return b, nil
									}
								} else {
									if b, exists := lm["sha256:"+digestStr]; exists {
										return b, nil
									}
								}
							}
						}
						// Content not present in report
						digest, _ := layerMap["digest"].(string)
						return nil, fmt.Errorf("metadata layer found (digest: %s) but content not accessible", digest)
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("meta.json layer not found in strict profile artifact")
}

// getAnnotation gets an annotation value with fallback.
func getAnnotation(report *VerificationReport, keys ...string) string {
	annotations, ok := report.Details["annotations"].(map[string]string)
	if !ok {
		return ""
	}

	for _, key := range keys {
		if value, exists := annotations[key]; exists && value != "" {
			return value
		}
	}

	return ""
}

// ValidateModuleMetadata validates module metadata against schema.
func ValidateModuleMetadata(meta []byte, profile Profile) error {
	var moduleMeta ModuleMeta
	if err := json.Unmarshal(meta, &moduleMeta); err != nil {
		return fmt.Errorf("invalid metadata JSON: %w", err)
	}

	// Basic validation
	if moduleMeta.Name == "" {
		return ValidationError{Field: "name", Message: "name is required"}
	}

	if moduleMeta.Version == "" {
		return ValidationError{Field: "version", Message: "version is required"}
	}

	// Validate version format
	if !isValidVersion(moduleMeta.Version) {
		return ValidationError{Field: "version", Message: "invalid version format"}
	}

	// Profile-specific validation
	switch profile {
	case ProfileStrict:
		// Additional strict validation
		if moduleMeta.Sum == "" {
			return ValidationError{Field: "sum", Message: "checksum is required in strict profile"}
		}
	}

	return nil
}

// isValidVersion checks if a version string is valid.
func isValidVersion(version string) bool {
	// Simple semver validation
	// In production, use a proper semver library
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	for _, part := range parts {
		// Check if it's a number (simplified)
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				// Allow pre-release versions like 1.0.0-beta
				if r == '-' && strings.Contains(version, "-") {
					break
				}
				return false
			}
		}
	}

	return true
}

// GetShapeRules returns shape validation rules for a profile.
func GetShapeRules(profile Profile) ShapeRules {
	switch profile {
	case ProfileCompat:
		return ShapeRules{
			RequiredLayers: []LayerRule{
				{
					MediaType: "application/vnd.oci.image.layer.v1.tar",
					Index:     0,
				},
			},
			ArtifactType: "application/vnd.oci.image.layer.v1.tar",
		}
	case ProfileStrict:
		return ShapeRules{
			RequiredLayers: []LayerRule{
				{
					MediaType: "application/vnd.projectcatalyst.kcl.module.tar.v1",
					Index:     0,
				},
				{
					MediaType: "application/vnd.projectcatalyst.kcl.module.meta.v1+json",
					Index:     1,
				},
			},
			ArtifactType: "application/vnd.projectcatalyst.kcl.module.v1+tar",
		}
	default:
		return ShapeRules{}
	}
}

// ShapeRules defines shape validation rules.
type ShapeRules struct {
	RequiredLayers []LayerRule
	ArtifactType   string
}

// LayerRule defines a required layer.
type LayerRule struct {
	MediaType string
	Index     int
}

// ValidateManifest validates an OCI manifest against shape rules.
func ValidateManifest(manifest ocispec.Manifest, rules ShapeRules) error {
	// Check layer count
	if len(manifest.Layers) != len(rules.RequiredLayers) {
		return fmt.Errorf("expected %d layers, got %d", len(rules.RequiredLayers), len(manifest.Layers))
	}

	// Check each layer
	for _, rule := range rules.RequiredLayers {
		if rule.Index >= len(manifest.Layers) {
			return fmt.Errorf("missing layer at index %d", rule.Index)
		}

		layer := manifest.Layers[rule.Index]
		if layer.MediaType != rule.MediaType {
			return fmt.Errorf("layer %d: expected media type %s, got %s",
				rule.Index, rule.MediaType, layer.MediaType)
		}
	}

	return nil
}

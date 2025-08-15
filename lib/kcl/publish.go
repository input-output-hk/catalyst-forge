package kcl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/input-output-hk/catalyst-forge/lib/kcl/internal"
	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Publish publishes a KCL module to an OCI registry.
func Publish(ctx context.Context, ociCli OCI, opts PublishOptions) (string, error) {
	// Validate options
	if err := validatePublishOptions(opts); err != nil {
		return "", fmt.Errorf("invalid publish options: %w", err)
	}

	// Read module metadata
	moduleMeta, err := readModuleMetadata(opts.ModuleRoot)
	if err != nil {
		return "", fmt.Errorf("failed to read module metadata: %w", err)
	}

	// Pack the module
	packOpts := DefaultPackOptions()
	tarBytes, checksum, err := PackModule(opts.ModuleRoot, packOpts)
	if err != nil {
		return "", fmt.Errorf("failed to pack module: %w", err)
	}

	// Build OCI artifact based on profile
	var manifest ocispec.Manifest
	var layers []ocispec.Descriptor
	var artifactType string

	switch opts.Profile {
	case ProfileCompat:
		manifest, layers, artifactType, err = buildCompatArtifact(tarBytes, checksum, moduleMeta, opts)
	case ProfileStrict:
		manifest, layers, artifactType, err = buildStrictArtifact(tarBytes, checksum, moduleMeta, opts)
	default:
		return "", fmt.Errorf("unsupported profile: %s", opts.Profile)
	}

	if err != nil {
		return "", fmt.Errorf("failed to build artifact: %w", err)
	}

	// Build annotations
	annotations := buildAnnotations(moduleMeta, checksum, opts)

	// Push to registry
	digest, err := ociCli.PushArtifact(opts.Ref, layers, manifest, artifactType, annotations)
	if err != nil {
		return "", fmt.Errorf("failed to push artifact: %w", err)
	}

	// Sign if requested
	if opts.Sign {
		if opts.SignKeyRef == "" {
			return "", fmt.Errorf("signing key reference required when signing is enabled")
		}
		if err := ociCli.SignArtifact(digest, opts.SignKeyRef); err != nil {
			return "", fmt.Errorf("failed to sign artifact: %w", err)
		}
	}

	// Attest if requested
	if opts.Attest {
		if len(opts.AttestBytes) == 0 {
			// Generate default SLSA attestation
			opts.AttestBytes, err = generateDefaultAttestation(digest, moduleMeta, checksum)
			if err != nil {
				return "", fmt.Errorf("failed to generate attestation: %w", err)
			}
		}

		_, err := ociCli.AttestArtifact(digest, "application/vnd.in-toto+json", opts.AttestBytes)
		if err != nil {
			return "", fmt.Errorf("failed to attest artifact: %w", err)
		}
	}

	return digest, nil
}

// validatePublishOptions validates publish options.
func validatePublishOptions(opts PublishOptions) error {
	if opts.ModuleRoot == "" {
		return fmt.Errorf("module root is required")
	}

	if opts.Ref == "" {
		return fmt.Errorf("reference is required")
	}

	if opts.Profile == "" {
		opts.Profile = ProfileCompat
	}

	if opts.Profile != ProfileCompat && opts.Profile != ProfileStrict {
		return fmt.Errorf("invalid profile: %s", opts.Profile)
	}

	if opts.Sign && opts.SignKeyRef == "" {
		return fmt.Errorf("sign key reference required when signing is enabled")
	}

	return nil
}

// readModuleMetadata reads metadata from kcl.mod file.
func readModuleMetadata(moduleRoot string) (*ModuleMeta, error) {
	kclModPath := filepath.Join(moduleRoot, "kcl.mod")

	// Read kcl.mod file
	content, err := os.ReadFile(kclModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kcl.mod: %w", err)
	}

	// Parse kcl.mod (simplified TOML-like format)
	meta := &ModuleMeta{}
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		switch key {
		case "name":
			meta.Name = value
		case "version":
			meta.Version = value
		case "description":
			meta.Description = value
		case "authors":
			// Handle array syntax
			meta.Authors = parseArray(value)
		case "license":
			meta.License = value
		case "repository":
			meta.Repository = value
		case "homepage":
			meta.Homepage = value
		}
	}

	// Validate required fields
	if meta.Name == "" {
		return nil, fmt.Errorf("module name is required in kcl.mod")
	}
	if meta.Version == "" {
		return nil, fmt.Errorf("module version is required in kcl.mod")
	}

	return meta, nil
}

// parseArray parses a simple array from kcl.mod.
func parseArray(value string) []string {
	// Handle ["item1", "item2"] format
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")

	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "\"'")
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

// buildCompatArtifact builds a KPM-compatible OCI artifact.
func buildCompatArtifact(tarBytes []byte, checksum string, _ *ModuleMeta, _ PublishOptions) (ocispec.Manifest, []ocispec.Descriptor, string, error) {
	// Create single tar layer
	tarLayer := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.layer.v1.tar",
		Digest:    digest.Digest("sha256:" + checksum),
		Size:      int64(len(tarBytes)),
	}

	// Create minimal config JSON and compute its digest/size
	configJSON := []byte(`{"created":"0001-01-01T00:00:00Z","type":"kcl-compat"}`)
	configDigest := digest.Digest("sha256:" + internal.HashBytes(configJSON))
	configSize := int64(len(configJSON))

	// Create manifest
	manifest := ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageManifest,
		Config: ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    configDigest,
			Size:      configSize,
		},
		Layers: []ocispec.Descriptor{tarLayer},
	}

	artifactType := "application/vnd.oci.image.layer.v1.tar"

	return manifest, []ocispec.Descriptor{tarLayer}, artifactType, nil
}

// buildStrictArtifact builds a Forge-specific OCI artifact.
func buildStrictArtifact(tarBytes []byte, checksum string, meta *ModuleMeta, opts PublishOptions) (ocispec.Manifest, []ocispec.Descriptor, string, error) {
	// Enhance metadata with strict profile information
	strictMeta := enhanceMetadataForStrict(meta, checksum, opts)

	// Create tar layer
	tarLayer := ocispec.Descriptor{
		MediaType: "application/vnd.projectcatalyst.kcl.module.tar.v1",
		Digest:    digest.Digest("sha256:" + checksum),
		Size:      int64(len(tarBytes)),
		Annotations: map[string]string{
			"org.opencontainers.image.title": "module.tar",
		},
	}

	// Create metadata JSON with validation
	metaJSON, err := json.MarshalIndent(strictMeta, "", "  ")
	if err != nil {
		return ocispec.Manifest{}, nil, "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Validate metadata against CUE schema
	if err := validateStrictMetadata(metaJSON); err != nil {
		return ocispec.Manifest{}, nil, "", fmt.Errorf("metadata validation failed: %w", err)
	}

	metaChecksum := internal.HashBytes(metaJSON)
	metaLayer := ocispec.Descriptor{
		MediaType: "application/vnd.projectcatalyst.kcl.module.meta.v1+json",
		Digest:    digest.Digest("sha256:" + metaChecksum),
		Size:      int64(len(metaJSON)),
		Annotations: map[string]string{
			"org.opencontainers.image.title": "meta.json",
		},
	}

	// Create manifest
	manifest := ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageManifest,
		Config: ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    digest.Digest("sha256:" + metaChecksum),
			Size:      int64(len(metaJSON)),
		},
		Layers: []ocispec.Descriptor{tarLayer, metaLayer},
	}

	artifactType := "application/vnd.projectcatalyst.kcl.module.v1+tar"

	return manifest, []ocispec.Descriptor{tarLayer, metaLayer}, artifactType, nil
}

// buildAnnotations builds OCI annotations for the artifact.
func buildAnnotations(meta *ModuleMeta, checksum string, opts PublishOptions) map[string]string {
	annotations := make(map[string]string)

	// Standard OCI annotations
	annotations["org.opencontainers.image.title"] = meta.Name
	annotations["org.opencontainers.image.version"] = meta.Version

	if meta.Description != "" {
		annotations["org.opencontainers.image.description"] = meta.Description
	}

	if len(meta.Authors) > 0 {
		annotations["org.opencontainers.image.authors"] = strings.Join(meta.Authors, ", ")
	}

	if meta.License != "" {
		annotations["org.opencontainers.image.licenses"] = meta.License
	}

	if meta.Repository != "" {
		annotations["org.opencontainers.image.source"] = meta.Repository
	}

	if meta.Homepage != "" {
		annotations["org.opencontainers.image.url"] = meta.Homepage
	}

	// KCL/KPM specific annotations
	annotations["io.kcl.name"] = meta.Name
	annotations["io.kcl.version"] = meta.Version
	annotations["io.kcl.sum"] = checksum

	// Add custom annotations
	for k, v := range opts.Annotations {
		annotations[k] = v
	}

	return annotations
}

// generateDefaultAttestation generates a default SLSA attestation.
func generateDefaultAttestation(digest string, meta *ModuleMeta, checksum string) ([]byte, error) {
	// Simplified SLSA predicate
	predicate := map[string]interface{}{
		"buildType": "https://kcl-lang.io/slsa/v1",
		"builder": map[string]string{
			"id": "forge-kcl-publisher",
		},
		"invocation": map[string]interface{}{
			"configSource": map[string]interface{}{
				"uri":    meta.Repository,
				"digest": map[string]interface{}{"sha256": checksum},
			},
		},
		"metadata": map[string]interface{}{
			"completeness": map[string]bool{
				"parameters": true,
				"materials":  true,
			},
			"reproducible": true,
		},
		"materials": []map[string]interface{}{
			{
				"uri": meta.Repository,
				"digest": map[string]interface{}{
					"sha256": checksum,
				},
			},
		},
		"subject": []map[string]interface{}{
			{
				"name": meta.Name,
				"digest": map[string]interface{}{
					"sha256": strings.TrimPrefix(digest, "sha256:"),
				},
			},
		},
	}

	// Create DSSE envelope
	envelope := map[string]interface{}{
		"payloadType": "application/vnd.in-toto+json",
		"payload":     predicate,
		"signatures":  []interface{}{}, // Will be populated by signing
	}

	return json.Marshal(envelope)
}

// enhanceMetadataForStrict adds strict profile specific metadata
func enhanceMetadataForStrict(meta *ModuleMeta, checksum string, opts PublishOptions) *ModuleMeta {
	// Clone the metadata
	enhanced := *meta

	// Add checksum
	enhanced.Sum = checksum

	// Add entry point if not set
	if enhanced.Entry == "" {
		// Check for common entry points
		if fileExists(filepath.Join(opts.ModuleRoot, "main.k")) {
			enhanced.Entry = "main.k"
		} else if fileExists(filepath.Join(opts.ModuleRoot, "index.k")) {
			enhanced.Entry = "index.k"
		}
	}

	// Add Forge-specific annotations
	if enhanced.Annotations == nil {
		enhanced.Annotations = make(map[string]string)
	}
	enhanced.Annotations["dev.catalyst.forge.profile"] = "strict"
	enhanced.Annotations["dev.catalyst.forge.version"] = "1.0.0"

	// Merge with user-provided annotations
	for k, v := range opts.Annotations {
		enhanced.Annotations[k] = v
	}

	return &enhanced
}

// validateStrictMetadata validates metadata against the CUE schema
func validateStrictMetadata(metaJSON []byte) error {
	// TODO: Integrate with lib/ociv2/validate/cue for actual CUE validation
	// For now, perform basic validation

	var meta map[string]interface{}
	if err := json.Unmarshal(metaJSON, &meta); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Check required fields
	requiredFields := []string{"name", "version", "sum"}
	for _, field := range requiredFields {
		if _, exists := meta[field]; !exists {
			return fmt.Errorf("required field %q is missing", field)
		}
	}

	// Validate name format
	if name, ok := meta["name"].(string); ok {
		if !isValidModuleName(name) {
			return fmt.Errorf("invalid module name format: %s", name)
		}
	}

	// Validate version format
	if version, ok := meta["version"].(string); ok {
		if !isValidSemver(version) {
			return fmt.Errorf("invalid version format: %s", version)
		}
	}

	// Validate sum format
	if sum, ok := meta["sum"].(string); ok {
		if !strings.HasPrefix(sum, "sha256:") || len(sum) != 71 {
			return fmt.Errorf("invalid sum format: %s", sum)
		}
	}

	return nil
}

// isValidModuleName checks if a module name is valid
func isValidModuleName(name string) bool {
	if name == "" || len(name) > 100 {
		return false
	}
	// Must start with letter, can contain letters, numbers, hyphens
	for i, ch := range name {
		if i == 0 {
			if !isLetter(ch) {
				return false
			}
		} else {
			if !isLetter(ch) && !isDigit(ch) && ch != '-' && ch != '_' {
				return false
			}
		}
	}
	return true
}

// isValidSemver checks if a version string is valid semver
func isValidSemver(version string) bool {
	// Simple semver validation
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return false
	}
	// Check major, minor, patch are numbers
	for i := 0; i < 3; i++ {
		if i < len(parts) {
			// Handle pre-release versions
			part := strings.Split(parts[i], "-")[0]
			for _, ch := range part {
				if !isDigit(ch) {
					return false
				}
			}
		}
	}
	return true
}

// Helper functions for character classification
func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

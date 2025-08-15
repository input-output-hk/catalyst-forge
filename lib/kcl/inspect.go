package kcl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	
	"github.com/input-output-hk/catalyst-forge/lib/kcl/cache"
)

// InspectResult contains the inspection result of a KCL module.
type InspectResult struct {
	Digest      string                 `json:"digest"`
	Profile     Profile                `json:"profile"`
	Metadata    *ModuleMeta            `json:"metadata"`
	Manifest    map[string]interface{} `json:"manifest,omitempty"`
	Layers      []LayerInfo            `json:"layers"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Signed      bool                   `json:"signed"`
	CreatedAt   time.Time              `json:"createdAt,omitempty"`
}

// LayerInfo contains information about a layer.
type LayerInfo struct {
	MediaType   string `json:"mediaType"`
	Digest      string `json:"digest"`
	Size        int64  `json:"size"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Inspect inspects a KCL module without pulling/extracting it.
func Inspect(ctx context.Context, ociCli OCI, ref ModuleRef, opts InspectOptions) (*InspectResult, error) {
	// Build reference string
	refStr := buildReference(ref)
	
	// Perform verification to get metadata
	report, err := ociCli.VerifyArtifact(refStr, opts.RequireSignature)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect artifact: %w", err)
	}
	
	// Check signature if required
	if opts.RequireSignature && !report.SignatureValid {
		return nil, ErrSignatureInvalid
	}
	
	// Determine profile from artifact shape
	profile := detectProfile(report)
	
	// Extract metadata based on profile
	metaBytes, err := extractMetadata(report, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}
	
	var meta ModuleMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	
	// Build inspection result
	result := &InspectResult{
		Digest:   report.Digest,
		Profile:  profile,
		Metadata: &meta,
		Signed:   report.SignatureValid,
		Layers:   extractLayerInfo(report),
	}
	
	// Add annotations if available
	if annotations, ok := report.Details["annotations"].(map[string]string); ok {
		result.Annotations = annotations
	}
	
	// Add manifest if available
	if manifest, ok := report.Details["manifest"].(map[string]interface{}); ok {
		result.Manifest = manifest
	}
	
	// Extract created time if available
	if createdStr, ok := result.Annotations["org.opencontainers.image.created"]; ok {
		if created, err := time.Parse(time.RFC3339, createdStr); err == nil {
			result.CreatedAt = created
		}
	}
	
	return result, nil
}

// detectProfile detects the profile from artifact shape.
func detectProfile(report *VerificationReport) Profile {
	if report.Details == nil {
		return ProfileCompat // Default to compat
	}
	
	// Check artifact type
	if artifactType, ok := report.Details["artifactType"].(string); ok {
		if strings.Contains(artifactType, "projectcatalyst") {
			return ProfileStrict
		}
	}
	
	// Check layer count and media types
	if manifest, ok := report.Details["manifest"].(map[string]interface{}); ok {
		if layers, ok := manifest["layers"].([]interface{}); ok {
			if len(layers) == 2 {
				// Check for strict profile media types
				for _, layer := range layers {
					if layerMap, ok := layer.(map[string]interface{}); ok {
						mediaType, _ := layerMap["mediaType"].(string)
						if strings.Contains(mediaType, "projectcatalyst.kcl.module") {
							return ProfileStrict
						}
					}
				}
			}
		}
	}
	
	// Check annotations
	if annotations, ok := report.Details["annotations"].(map[string]string); ok {
		if profile, exists := annotations["dev.catalyst.forge.profile"]; exists {
			switch profile {
			case "strict":
				return ProfileStrict
			case "compat":
				return ProfileCompat
			}
		}
	}
	
	return ProfileCompat // Default to compat
}

// extractLayerInfo extracts layer information from verification report.
func extractLayerInfo(report *VerificationReport) []LayerInfo {
	var layers []LayerInfo
	
	if report.Details == nil {
		return layers
	}
	
	// Extract from manifest
	if manifest, ok := report.Details["manifest"].(map[string]interface{}); ok {
		if manifestLayers, ok := manifest["layers"].([]interface{}); ok {
			for _, layer := range manifestLayers {
				if layerMap, ok := layer.(map[string]interface{}); ok {
					info := LayerInfo{
						MediaType: getString(layerMap, "mediaType"),
						Digest:    getString(layerMap, "digest"),
						Size:      getInt64(layerMap, "size"),
					}
					
					if annotations, ok := layerMap["annotations"].(map[string]string); ok {
						info.Annotations = annotations
					}
					
					layers = append(layers, info)
				}
			}
		}
	}
	
	return layers
}

// InspectLocal inspects a locally cached module.
func InspectLocal(ctx context.Context, digest string) (*InspectResult, error) {
	cm, err := cache.GetManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache manager: %w", err)
	}
	
	// Check if module exists in cache
	moduleDir := cm.ModulePath(stripDigestPrefix(digest))
	metaPath := filepath.Join(moduleDir, ".meta.json")
	
	if !fileExists(metaPath) {
		return nil, fmt.Errorf("module not found in cache: %s", digest)
	}
	
	// Read metadata
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}
	
	var meta ModuleMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	
	// Determine profile from metadata
	profile := ProfileCompat
	if meta.Annotations != nil {
		if p, exists := meta.Annotations["dev.catalyst.forge.profile"]; exists && p == "strict" {
			profile = ProfileStrict
		}
	}
	
	// Build inspection result
	result := &InspectResult{
		Digest:      digest,
		Profile:     profile,
		Metadata:    &meta,
		Annotations: meta.Annotations,
	}
	
	// Get cache stats
	stampPath := filepath.Join(moduleDir, ".stamp")
	if info, err := os.Stat(stampPath); err == nil {
		result.CreatedAt = info.ModTime()
	}
	
	return result, nil
}

// Helper functions

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt64(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	}
	return 0
}


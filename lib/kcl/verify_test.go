package kcl

import (
	"encoding/json"
	"testing"
)

func TestExtractStrictMetadata_FromMetaLayer(t *testing.T) {
	meta := ModuleMeta{
		Name:    "test",
		Version: "1.0.0",
		Sum:     "sha256:abc",
	}
	metaBytes, _ := json.Marshal(meta)

	report := &VerificationReport{
		Details: map[string]interface{}{
			"metaLayer": metaBytes,
		},
	}

	out, err := extractStrictMetadata(report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var got ModuleMeta
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if got.Name != meta.Name || got.Version != meta.Version || got.Sum != meta.Sum {
		t.Fatalf("unexpected meta: %+v", got)
	}
}

func TestExtractStrictMetadata_FromManifestAndLayers(t *testing.T) {
	meta := ModuleMeta{
		Name:    "mod",
		Version: "0.1.0",
		Sum:     "sha256:def",
	}
	metaBytes, _ := json.Marshal(meta)

	// The layer digest we'll advertise in the manifest
	layerDigest := "sha256:deadbeef"

	report := &VerificationReport{
		Details: map[string]interface{}{
			"manifest": map[string]interface{}{
				"layers": []interface{}{
					map[string]interface{}{
						"mediaType": "application/vnd.projectcatalyst.kcl.module.tar.v1",
						"digest":    "sha256:aaaa",
						"size":      1,
					},
					map[string]interface{}{
						"mediaType": "application/vnd.projectcatalyst.kcl.module.meta.v1+json",
						"digest":    layerDigest,
						"size":      len(metaBytes),
					},
				},
			},
			// Layers map keyed by unprefixed digest
			"layers": map[string][]byte{
				"deadbeef": metaBytes,
			},
		},
	}

	out, err := extractStrictMetadata(report)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var got ModuleMeta
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if got.Name != meta.Name || got.Version != meta.Version || got.Sum != meta.Sum {
		t.Fatalf("unexpected meta: %+v", got)
	}
}

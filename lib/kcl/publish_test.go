package kcl

import (
	"encoding/json"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestBuildCompatArtifact_ConfigDescriptor(t *testing.T) {
	tar := []byte{1, 2, 3}
	checksum := "abcd"

	manifest, layers, artifactType, err := buildCompatArtifact(tar, checksum, nil, PublishOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if artifactType != "application/vnd.oci.image.layer.v1.tar" {
		t.Fatalf("unexpected artifact type: %s", artifactType)
	}
	if len(layers) != 1 {
		t.Fatalf("expected 1 layer, got %d", len(layers))
	}
	if manifest.Config.MediaType != "application/vnd.oci.image.config.v1+json" {
		t.Fatalf("unexpected config mediaType: %s", manifest.Config.MediaType)
	}
	if string(manifest.Config.Digest) == "sha256:"+checksum || manifest.Config.Size == 0 {
		t.Fatalf("config digest/size should be real bytes, got digest=%s size=%d", manifest.Config.Digest, manifest.Config.Size)
	}
}

func TestBuildStrictArtifact_IncludesMetaLayer(t *testing.T) {
	tar := []byte{9, 9, 9}
	checksum := "feed"
	meta := &ModuleMeta{Name: "x", Version: "1.0.0", Sum: "sha256:zzz"}

	manifest, layers, artifactType, err := buildStrictArtifact(tar, checksum, meta, PublishOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if artifactType == "" || len(layers) != 2 {
		t.Fatalf("expected 2 layers and non-empty type, got %d %s", len(layers), artifactType)
	}

	// Basic sanity: layers include meta.json media type
	hasMeta := false
	for _, l := range manifest.Layers {
		if l.MediaType == "application/vnd.projectcatalyst.kcl.module.meta.v1+json" {
			hasMeta = true
			break
		}
	}
	if !hasMeta {
		b, _ := json.MarshalIndent(manifest, "", "  ")
		t.Fatalf("expected meta layer in manifest, got: %s", string(b))
	}

	// Config should be JSON
	if manifest.Config.MediaType != ocispec.MediaTypeImageConfig {
		t.Fatalf("unexpected config mediaType: %s", manifest.Config.MediaType)
	}
}

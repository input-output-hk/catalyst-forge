package kcl

import (
	"encoding/json"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestExtractTarLayer_Compat(t *testing.T) {
	data := []byte{1, 2, 3}
	res := &OCIPullResult{
		Manifest: ocispec.Manifest{
			Layers: []ocispec.Descriptor{
				{MediaType: "application/vnd.oci.image.layer.v1.tar", Digest: "sha256:abcd", Size: int64(len(data))},
			},
		},
		Layers: map[string][]byte{
			"abcd": data,
		},
	}

	out, err := extractTarLayer(res, ProfileCompat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != string(data) {
		t.Fatalf("unexpected bytes")
	}
}

func TestExtractTarLayer_Strict(t *testing.T) {
	data := []byte{7, 7}
	res := &OCIPullResult{
		Manifest: ocispec.Manifest{
			Layers: []ocispec.Descriptor{
				{MediaType: "application/vnd.projectcatalyst.kcl.module.tar.v1", Digest: "sha256:beef", Size: int64(len(data))},
			},
		},
		Layers: map[string][]byte{
			"beef": data,
		},
	}

	out, err := extractTarLayer(res, ProfileStrict)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != string(data) {
		t.Fatalf("unexpected bytes")
	}
}

func TestGenerateMetadataFromPull_Compat(t *testing.T) {
	res := &OCIPullResult{
		Annotations: map[string]string{
			"io.kcl.name":    "my-mod",
			"io.kcl.version": "1.0.0",
			"io.kcl.sum":     "sha256:abc",
		},
	}
	b, err := generateMetadataFromPull(res, ProfileCompat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var meta ModuleMeta
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if meta.Name != "my-mod" || meta.Version != "1.0.0" || meta.Sum != "sha256:abc" {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestGenerateMetadataFromPull_Strict(t *testing.T) {
	meta := ModuleMeta{Name: "s", Version: "0.0.1", Sum: "sha256:zzz"}
	metaBytes, _ := json.Marshal(meta)
	res := &OCIPullResult{
		Manifest: ocispec.Manifest{
			Layers: []ocispec.Descriptor{
				{MediaType: "application/vnd.projectcatalyst.kcl.module.tar.v1", Digest: "sha256:aaaa"},
				{MediaType: "application/vnd.projectcatalyst.kcl.module.meta.v1+json", Digest: "sha256:deadc0de", Size: int64(len(metaBytes))},
			},
		},
		Layers: map[string][]byte{
			// Unprefixed digest key should be accepted
			"deadc0de": metaBytes,
		},
	}
	b, err := generateMetadataFromPull(res, ProfileStrict)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got ModuleMeta
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != meta.Name || got.Version != meta.Version || got.Sum != meta.Sum {
		t.Fatalf("unexpected meta: %+v", got)
	}
}

package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	ociv2 "github.com/input-output-hk/catalyst-forge/lib/ociv2"
)

// helpers
func digestHex(d string) string {
	return strings.TrimPrefix(d, "sha256:")
}

// CompatSelector builds the JSON expected by #KCLCompatValidation
type CompatSelector struct{}

func (CompatSelector) Extract(ctx context.Context, pr *ociv2.PullResult) ([]byte, error) {
	if len(pr.Layers) != 1 {
		return nil, fmt.Errorf("compat expects exactly 1 layer, got %d", len(pr.Layers))
	}
	layer := pr.Layers[0]
	// read layer digest hex from descriptor
	hex := digestHex(layer.Digest)

	manifest := map[string]interface{}{
		"artifactType": pr.ArtifactType,
		"annotations":  pr.ManifestAnn,
		"layers": []map[string]interface{}{
			{
				"mediaType": layer.MediaType,
				"digest":    layer.Digest,
				"size":      layer.Size,
			},
		},
	}

	doc := map[string]interface{}{
		"manifest": manifest,
		"tarHex":   hex,
		"layerHex": hex,
	}
	return json.Marshal(doc)
}

// StrictMetaSelector returns the strict meta.json layer content
type StrictMetaSelector struct{}

func (StrictMetaSelector) Extract(ctx context.Context, pr *ociv2.PullResult) ([]byte, error) {
	// find meta layer by media type
	var meta *ociv2.PulledLayer
	for _, l := range pr.Layers {
		if l.MediaType == "application/vnd.projectcatalyst.kcl.module.meta.v1+json" {
			ll := l // copy
			meta = &ll
			break
		}
	}
	if meta == nil {
		return nil, fmt.Errorf("meta layer not found")
	}
	rc, err := meta.Open()
	if err != nil {
		return nil, err
	}
	b, rerr := io.ReadAll(rc)
	cerr := rc.Close()
	if rerr != nil {
		return nil, rerr
	}
	if cerr != nil {
		return nil, cerr
	}
	return b, nil
}

// StrictInputSelector builds the JSON expected by #KCLStrictValidation
type StrictInputSelector struct{}

func (StrictInputSelector) Extract(ctx context.Context, pr *ociv2.PullResult) ([]byte, error) {
	if len(pr.Layers) != 2 {
		return nil, fmt.Errorf("strict expects exactly 2 layers, got %d", len(pr.Layers))
	}
	// Identify tar + meta
	var tar, meta *ociv2.PulledLayer
	for i := range pr.Layers {
		l := pr.Layers[i]
		switch l.MediaType {
		case "application/vnd.projectcatalyst.kcl.module.tar.v1":
			tar = &l
		case "application/vnd.projectcatalyst.kcl.module.meta.v1+json":
			meta = &l
		}
	}
	if tar == nil || meta == nil {
		return nil, fmt.Errorf("strict layers not found (tar=%v meta=%v)", tar != nil, meta != nil)
	}

	// project manifest
	mLayers := make([]map[string]interface{}, 0, 2)
	for _, l := range pr.Layers {
		mLayers = append(mLayers, map[string]interface{}{
			"mediaType": l.MediaType,
			"digest":    l.Digest,
			"size":      l.Size,
		})
	}
	manifest := map[string]interface{}{
		"artifactType": pr.ArtifactType,
		"annotations":  pr.ManifestAnn,
		"layers":       mLayers,
	}

	// read meta bytes
	rc, err := meta.Open()
	if err != nil {
		return nil, err
	}
	metaBytes, rerr := io.ReadAll(rc)
	cerr := rc.Close()
	if rerr != nil {
		return nil, rerr
	}
	if cerr != nil {
		return nil, cerr
	}

	doc := map[string]interface{}{
		"manifest": manifest,
		"meta":     json.RawMessage(metaBytes),
		"tarHex":   digestHex(tar.Digest),
	}
	return json.Marshal(doc)
}

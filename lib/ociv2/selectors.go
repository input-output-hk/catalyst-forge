package ociv2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// JSONFromConfig returns a selector that extracts the config JSON
func JSONFromConfig() JSONSelector {
	return jsonSelectorFunc(func(ctx context.Context, pr *PullResult) ([]byte, error) {
		if pr.Config == nil {
			return nil, fmt.Errorf("artifact has no config")
		}
		return pr.Config, nil
	})
}

// JSONFromLayerByMediaType returns a selector that extracts the first layer with matching media type
func JSONFromLayerByMediaType(mt string) JSONSelector {
	return jsonSelectorFunc(func(ctx context.Context, pr *PullResult) ([]byte, error) {
		for _, layer := range pr.Layers {
			if layer.MediaType == mt {
				// Open the layer and read its content
				rc, err := layer.Open()
				if err != nil {
					return nil, fmt.Errorf("failed to open layer: %w", err)
				}
				defer func() { _ = rc.Close() }()

				// Read all content
				data, err := io.ReadAll(rc)
				if err != nil {
					return nil, fmt.Errorf("failed to read layer: %w", err)
				}

				return data, nil
			}
		}
		return nil, fmt.Errorf("no layer found with media type %s", mt)
	})
}

// JSONFromLayerByIndex returns a selector that extracts a specific layer by index
func JSONFromLayerByIndex(i int) JSONSelector {
	return jsonSelectorFunc(func(ctx context.Context, pr *PullResult) ([]byte, error) {
		if i < 0 || i >= len(pr.Layers) {
			return nil, fmt.Errorf("layer index %d out of bounds (have %d layers)", i, len(pr.Layers))
		}

		layer := pr.Layers[i]
		rc, err := layer.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open layer %d: %w", i, err)
		}
		defer func() { _ = rc.Close() }()

		// Read all content
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("failed to read layer %d: %w", i, err)
		}

		return data, nil
	})
}

// JSONFromManifestAnnotations returns a selector that extracts annotations as JSON
func JSONFromManifestAnnotations(allowKeys []string) JSONSelector {
	return jsonSelectorFunc(func(ctx context.Context, pr *PullResult) ([]byte, error) {
		if pr.ManifestAnn == nil {
			return nil, fmt.Errorf("artifact has no manifest annotations")
		}

		// If allowKeys is specified, filter annotations
		result := make(map[string]string)
		if len(allowKeys) > 0 {
			allowed := make(map[string]bool)
			for _, k := range allowKeys {
				allowed[k] = true
			}
			for k, v := range pr.ManifestAnn {
				if allowed[k] {
					result[k] = v
				}
			}
		} else {
			result = pr.ManifestAnn
		}

		// Convert to JSON
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal annotations to JSON: %w", err)
		}

		return data, nil
	})
}

// jsonSelectorFunc is a helper type to convert a function to JSONSelector
type jsonSelectorFunc func(context.Context, *PullResult) ([]byte, error)

func (f jsonSelectorFunc) Extract(ctx context.Context, pr *PullResult) ([]byte, error) {
	return f(ctx, pr)
}

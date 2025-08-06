package helm

import (
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/external/helm"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/executor"
)

type HelmManifestGenerator struct {
	client helm.Client
	logger *slog.Logger
}

func (h *HelmManifestGenerator) Generate(mod sp.Module, raw cue.Value, env string) ([]byte, error) {
	values, ok := mod.Values.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to assert mod.Values to map[string]interface{}")
	}

	config := helm.TemplateConfig{
		ReleaseName: mod.Instance,
		Namespace:   mod.Namespace,
		ChartURL:    mod.Registry,
		ChartName:   mod.Name,
		Version:     mod.Version,
		Values:      values,
	}

	manifest, err := h.client.Template(config)
	if err != nil {
		return nil, fmt.Errorf("failed to template chart: %w", err)
	}

	return []byte(manifest), nil
}

func NewHelmManifestGenerator(logger *slog.Logger) (*HelmManifestGenerator, error) {
	if logger == nil {
		logger = slog.Default()
	}

	exec := executor.NewLocalExecutor(logger)
	client, err := helm.NewBinaryClient(exec, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Helm client: %w", err)
	}

	return &HelmManifestGenerator{
		client: client,
		logger: logger,
	}, nil
}

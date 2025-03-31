package helm

import (
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/providers/helm/downloader"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
)

type HelmManifestGenerator struct {
	downloader downloader.ChartDownloader
	logger     *slog.Logger
}

func (h *HelmManifestGenerator) Generate(mod sp.Module, raw cue.Value, env string) ([]byte, error) {
	client := action.NewInstall(&action.Configuration{})

	client.ReleaseName = mod.Instance
	client.Namespace = mod.Namespace

	client.ClientOnly = true
	client.DisableHooks = true
	client.DryRun = true
	client.IncludeCRDs = true

	data, err := h.downloader.Download(mod.Registry, mod.Name, mod.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to download chart: %w", err)
	}

	chart, err := loader.LoadArchive(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart archive: %w", err)
	}

	values, ok := mod.Values.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to assert mod.Values to map[string]interface{}")
	}

	rel, err := client.Run(chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to run client: %w", err)
	}

	return []byte(rel.Manifest), nil
}

func NewHelmManifestGenerator(logger *slog.Logger) *HelmManifestGenerator {
	return &HelmManifestGenerator{
		downloader: downloader.NewDefaultChartDownloader(logger),
		logger:     logger,
	}
}

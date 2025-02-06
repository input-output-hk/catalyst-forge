package helm

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/providers/helm/downloader"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
)

type HelmManifestGenerator struct {
	downloader downloader.ChartDownloader
	logger     *slog.Logger
}

func (h *HelmManifestGenerator) Generate(mod schema.DeploymentModule) ([]byte, error) {
	client := action.NewInstall(&action.Configuration{})

	client.ReleaseName = mod.Instance
	client.Namespace = mod.Namespace

	client.ClientOnly = true
	client.DisableHooks = true
	client.DryRun = true
	client.IncludeCRDs = true

	data, err := h.downloader.Download(*mod.Registry, *mod.Name, *mod.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to download chart: %w", err)
	}

	chart, err := loader.LoadArchive(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart archive: %w", err)
	}

	rel, err := client.Run(chart, map[string]interface{}{"foo": "bar"})
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

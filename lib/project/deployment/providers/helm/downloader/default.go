package downloader

import (
	"bytes"
	"fmt"
	"log/slog"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

// DefaultChartDownloader is a default implementation of ChartDownloader.
type DefaultChartDownloader struct {
	logger   *slog.Logger
	settings *cli.EnvSettings
}

func (d *DefaultChartDownloader) Download(repoUrl, chartName, version string) (*bytes.Buffer, error) {
	dl := downloader.ChartDownloader{
		Getters:          getter.All(d.settings),
		RepositoryConfig: d.settings.RepositoryConfig,
		RepositoryCache:  d.settings.RepositoryCache,
	}

	d.logger.Debug("Finding chart in repo", "repoUrl", repoUrl, "chartName", chartName, "version", version)
	url, err := repo.FindChartInRepoURL(repoUrl, chartName, "", "", "", "", getter.All(d.settings))
	if err != nil {
		return nil, fmt.Errorf("failed to find chart in repo: %w", err)
	}

	u, err := dl.ResolveChartVersion(url, version)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve chart version: %w", err)
	}

	d.logger.Debug("Downloading chart", "url", u.String())
	g, err := dl.Getters.ByScheme(u.Scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to get getter by scheme: %w", err)
	}

	return g.Get(u.String())
}

// NewDefaultChartDownloader creates a new DefaultChartDownloader.
func NewDefaultChartDownloader(logger *slog.Logger) *DefaultChartDownloader {
	return &DefaultChartDownloader{
		logger:   logger,
		settings: cli.New(),
	}
}

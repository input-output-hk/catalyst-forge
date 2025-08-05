package downloader

import "bytes"

//go:generate go run github.com/matryer/moq@latest -skip-ensure -pkg mocks -out mocks/downloader.go . ChartDownloader

// ChartDownloader is a downloader for Helm charts.
type ChartDownloader interface {
	// Download downloads a Helm chart from the given URL and version.
	Download(repoUrl, chartName, version string) (*bytes.Buffer, error)
}

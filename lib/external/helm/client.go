package helm

//go:generate go run github.com/matryer/moq@latest -skip-ensure -pkg mocks -out mocks/client.go . Client

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/input-output-hk/catalyst-forge/lib/tools/executor"
)

// Client is the interface for a Helm client.
type Client interface {
	Template(config TemplateConfig) (string, error)
}

// TemplateConfig contains the configuration for templating a Helm chart.
type TemplateConfig struct {
	ReleaseName string
	Namespace   string
	ChartURL    string
	ChartName   string
	Version     string
	Values      map[string]interface{}
}

// BinaryClient is a Helm client that uses the Helm binary via executor.
type BinaryClient struct {
	executor executor.WrappedExecuter
	logger   *slog.Logger
}

// NewBinaryClient creates a new BinaryClient.
// It ensures the Helm binary exists and returns an error if not found.
func NewBinaryClient(exec executor.Executor, logger *slog.Logger) (*BinaryClient, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Check if helm binary exists
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		return nil, fmt.Errorf("helm binary not found in PATH: %w", err)
	}

	logger.Debug("Found Helm binary", "path", helmPath)

	// Create wrapped executor for helm command
	wrappedExec := executor.NewWrappedLocalExecutor(exec, "helm")

	return &BinaryClient{
		executor: wrappedExec,
		logger:   logger,
	}, nil
}

// Template renders a Helm chart template and returns the manifest.
func (c *BinaryClient) Template(config TemplateConfig) (string, error) {
	// Create a temporary directory for chart download
	tempDir, err := os.MkdirTemp("", "helm-chart-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	chartPath := filepath.Join(tempDir, config.ChartName)

	// First, pull the chart
	if err := c.pullChart(config, chartPath); err != nil {
		return "", fmt.Errorf("failed to pull chart: %w", err)
	}

	// Then template the chart
	return c.templateChart(config, chartPath)
}

// pullChart downloads the Helm chart to the specified path.
func (c *BinaryClient) pullChart(config TemplateConfig, chartPath string) error {
	args := []string{"pull"}

	// Add repository URL and chart name
	chartRef := fmt.Sprintf("%s/%s", strings.TrimSuffix(config.ChartURL, "/"), config.ChartName)
	args = append(args, chartRef)

	// Add version if specified
	if config.Version != "" {
		args = append(args, "--version", config.Version)
	}

	// Extract to the temp directory
	args = append(args, "--untar", "--untardir", filepath.Dir(chartPath))

	c.logger.Debug("Pulling Helm chart", "chartRef", chartRef, "version", config.Version)

	output, err := c.executor.Execute(args...)
	if err != nil {
		c.logger.Error("Helm pull command failed", "args", args, "output", string(output), "error", err)
		return fmt.Errorf("failed to pull chart with args %v: %w\nOutput: %s", args, err, string(output))
	}

	return nil
}

// templateChart renders the Helm chart template.
func (c *BinaryClient) templateChart(config TemplateConfig, chartPath string) (string, error) {
	args := []string{"template", config.ReleaseName, chartPath}

	// Add namespace if specified
	if config.Namespace != "" {
		args = append(args, "--namespace", config.Namespace)
	}

	// Add values
	for key, value := range config.Values {
		args = append(args, "--set", fmt.Sprintf("%s=%v", key, value))
	}

	// Additional template options to match the original behavior
	args = append(args,
		"--include-crds",
		"--skip-tests",
	)

	c.logger.Debug("Templating Helm chart", "releaseName", config.ReleaseName, "chartPath", chartPath)

	output, err := c.executor.Execute(args...)
	if err != nil {
		c.logger.Error("Helm template command failed", "args", args, "output", string(output), "error", err)
		return "", fmt.Errorf("failed to template chart with args %v: %w\nOutput: %s", args, err, string(output))
	}

	return string(output), nil
}

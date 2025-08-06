package kcl

//go:generate go run github.com/matryer/moq@latest -skip-ensure -pkg mocks -out mocks/client.go . Client

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/input-output-hk/catalyst-forge/lib/oci"
	"github.com/input-output-hk/catalyst-forge/lib/tools/executor"
)

// Client is the interface for a KCL client.
type Client interface {
	Run(string, ModuleConfig) (string, error)
}

// BinaryClient is a KCL client that uses the KCL binary via executor.
type BinaryClient struct {
	executor  executor.WrappedExecuter
	logger    *slog.Logger
	cachePath string
	ociClient oci.Client
}

// Option configures the KCL client
type Option func(*BinaryClient) error

// WithCachePath sets the cache path for OCI modules
func WithCachePath(cachePath string) Option {
	return func(c *BinaryClient) error {
		if cachePath == "" {
			return fmt.Errorf("cache path cannot be empty")
		}
		c.cachePath = cachePath
		return nil
	}
}

// WithOCIClient sets a custom OCI client for downloading modules
func WithOCIClient(client oci.Client) Option {
	return func(c *BinaryClient) error {
		if client == nil {
			return fmt.Errorf("OCI client cannot be nil")
		}
		c.ociClient = client
		return nil
	}
}

// NewBinaryClient creates a new BinaryClient.
// It ensures the KCL binary exists and returns an error if not found.
func NewBinaryClient(exec executor.Executor, logger *slog.Logger, opts ...Option) (*BinaryClient, error) {
	if logger == nil {
		logger = slog.Default()
	}

	kclPath, err := exec.LookPath("kcl")
	if err != nil {
		return nil, fmt.Errorf("kcl binary not found in PATH: %w", err)
	}

	logger.Debug("Found KCL binary", "path", kclPath)

	wrappedExec := executor.NewWrappedLocalExecutor(exec, "kcl")

	client := &BinaryClient{
		executor: wrappedExec,
		logger:   logger,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Initialize OCI client if cache path is set but no custom client was provided
	if client.cachePath != "" && client.ociClient == nil {
		ociClient, err := oci.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create default OCI client: %w", err)
		}
		client.ociClient = ociClient
	}

	return client, nil
}

// Run executes a KCL module with the given configuration.
func (c *BinaryClient) Run(path string, conf ModuleConfig) (string, error) {
	var actualPath string

	// Check if this is an OCI path and handle accordingly
	if strings.HasPrefix(path, "oci://") {
		if c.ociClient == nil {
			return "", fmt.Errorf("OCI path provided (%s) but no OCI client configured", path)
		}
		if c.cachePath == "" {
			return "", fmt.Errorf("OCI path provided (%s) but no cache path configured", path)
		}

		cachedPath, err := c.cacheOCIModule(path)
		if err != nil {
			return "", fmt.Errorf("failed to cache OCI module: %w", err)
		}
		actualPath = cachedPath
		c.logger.Debug("Using cached OCI module", "original", path, "cached", actualPath)
	} else {
		actualPath = path
	}

	args, err := c.buildArgs(actualPath, conf)
	if err != nil {
		return "", fmt.Errorf("failed to build KCL arguments: %w", err)
	}

	output, err := c.executor.Execute(args...)
	if err != nil {
		c.logger.Error("KCL command failed", "args", args, "output", string(output), "error", err)
		return "", fmt.Errorf("failed to execute KCL command with args %v: %w\nOutput: %s", args, err, string(output))
	}

	return string(output), nil
}

// cacheOCIModule downloads and caches an OCI module, returning the local path
func (c *BinaryClient) cacheOCIModule(ociPath string) (string, error) {
	// Create a hash of the OCI path to use as a unique subdirectory
	hasher := sha256.New()
	hasher.Write([]byte(ociPath))
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	// Create the cache subdirectory path
	cacheDir := filepath.Join(c.cachePath, hashString)

	// Check if the module is already cached
	if _, err := os.Stat(cacheDir); err == nil {
		c.logger.Debug("OCI module already cached", "path", ociPath, "cache", cacheDir)
		return cacheDir, nil
	}

	c.logger.Info("Downloading OCI module to cache", "path", ociPath, "cache", cacheDir)

	// Strip the "oci://" prefix and parse the URL
	registryURL, err := c.parseOCIURL(ociPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse OCI URL %s: %w", ociPath, err)
	}

	c.logger.Debug("Parsed OCI URL", "original", ociPath, "parsed", registryURL)

	// Download the OCI module to the cache directory
	if err := c.ociClient.Pull(registryURL, cacheDir); err != nil {
		return "", fmt.Errorf("failed to download OCI module %s to %s: %w", ociPath, cacheDir, err)
	}

	c.logger.Debug("Successfully cached OCI module", "path", ociPath, "cache", cacheDir)
	return cacheDir, nil
}

// parseOCIURL converts an OCI URL with query parameters to standard Docker registry format
func (c *BinaryClient) parseOCIURL(ociPath string) (string, error) {
	// Strip the "oci://" prefix
	urlStr := strings.TrimPrefix(ociPath, "oci://")

	// Parse the URL to handle query parameters
	parsedURL, err := url.Parse("https://" + urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Get the base registry and repository path
	baseURL := parsedURL.Host + parsedURL.Path

	// Check for tag in query parameters
	if tag := parsedURL.Query().Get("tag"); tag != "" {
		return baseURL + ":" + tag, nil
	}

	// If no tag in query params, return as-is (may have tag in path already)
	return baseURL, nil
}

// buildArgs constructs the arguments for the kcl command.
func (c *BinaryClient) buildArgs(path string, conf ModuleConfig) ([]string, error) {
	args := []string{"run"}

	// The path should now always be a local path (either originally local or cached from OCI)
	args = append(args, path)

	configArgs, err := conf.ToArgs()
	if err != nil {
		return nil, fmt.Errorf("failed to convert config to arguments: %w", err)
	}
	args = append(args, configArgs...)

	return args, nil
}

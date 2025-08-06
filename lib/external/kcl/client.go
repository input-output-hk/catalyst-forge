package kcl

//go:generate go run github.com/matryer/moq@latest -skip-ensure -pkg mocks -out mocks/client.go . Client

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/input-output-hk/catalyst-forge/lib/tools/executor"
)

// Client is the interface for a KCL client.
type Client interface {
	Run(string, ModuleConfig) (string, error)
}

// BinaryClient is a KCL client that uses the KCL binary via executor.
type BinaryClient struct {
	executor executor.WrappedExecuter
	logger   *slog.Logger
}

// NewBinaryClient creates a new BinaryClient.
// It ensures the KCL binary exists and returns an error if not found.
func NewBinaryClient(exec executor.Executor, logger *slog.Logger) (*BinaryClient, error) {
	if logger == nil {
		logger = slog.Default()
	}

	kclPath, err := exec.LookPath("kcl")
	if err != nil {
		return nil, fmt.Errorf("kcl binary not found in PATH: %w", err)
	}

	logger.Debug("Found KCL binary", "path", kclPath)

	wrappedExec := executor.NewWrappedLocalExecutor(exec, "kcl")

	return &BinaryClient{
		executor: wrappedExec,
		logger:   logger,
	}, nil
}

// Run executes a KCL module with the given configuration.
func (c *BinaryClient) Run(path string, conf ModuleConfig) (string, error) {
	args, err := c.buildArgs(path, conf)
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

// buildArgs constructs the arguments for the kcl command.
func (c *BinaryClient) buildArgs(path string, conf ModuleConfig) ([]string, error) {
	args := []string{"run"}

	if strings.HasPrefix(path, "oci://") {
		args = append(args, path)
	} else {
		args = append(args, path)
	}

	configArgs, err := conf.ToArgs()
	if err != nil {
		return nil, fmt.Errorf("failed to convert config to arguments: %w", err)
	}
	args = append(args, configArgs...)

	return args, nil
}

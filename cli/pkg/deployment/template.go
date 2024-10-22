package deployment

import (
	"bytes"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	"github.com/spf13/afero"
)

// BundleTemplater is an interface for rendering a bundle template to YAML.
type BundleTemplater interface {
	Render(Bundle) (string, error)
}

type DefaultBundleTemplater struct {
	fs      afero.Fs
	logger  *slog.Logger
	timoni  executor.WrappedExecuter
	stdout  *bytes.Buffer
	stderr  *bytes.Buffer
	workdir string
}

func (t DefaultBundleTemplater) Render(bundle Bundle) (string, error) {
	t.logger.Info("Encoding bundle")
	src, err := bundle.Encode()
	if err != nil {
		return "", err
	}

	bundlePath := filepath.Join(t.workdir, "bundle.cue")
	t.logger.Info("Writing bundle", "path", bundlePath)
	if err := afero.WriteFile(t.fs, bundlePath, src, 0644); err != nil {
		return "", fmt.Errorf("could not write bundle: %w", err)
	}

	_, err = t.timoni.Execute("bundle", "build", "--log-pretty=false", "--log-color=false", "-f", bundlePath)
	if err != nil {
		t.logger.Error("Failed to build bundle", "error", err)
		t.logger.Error("Timoni output", "output", t.stderr.String())
		return "", fmt.Errorf("could not build bundle: %w", err)
	}

	return t.stdout.String(), nil
}

func NewDefaultBundleTemplater(logger *slog.Logger) (*DefaultBundleTemplater, error) {
	fs := afero.NewOsFs()
	workdir, err := afero.TempDir(fs, "", "catalyst-forge-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	var stdout, stderr bytes.Buffer

	return &DefaultBundleTemplater{
		fs:     afero.NewOsFs(),
		logger: logger,
		timoni: executor.NewLocalWrappedExecutor(
			executor.NewLocalExecutor(logger, executor.WithRedirectTo(&stdout, &stderr)),
			"timoni",
		),
		stdout:  &stdout,
		stderr:  &stderr,
		workdir: workdir,
	}, nil
}

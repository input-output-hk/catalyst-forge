package project

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/loader"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthfile"
	"github.com/spf13/afero"
)

// ProjectLoader is an interface for loading projects.
type ProjectLoader interface {
	// Load loads the project.
	Load() (Project, error)
}

// DefaultProjectLoader is the default implementation of the ProjectLoader.
type DefaultProjectLoader struct {
	blueprintLoader loader.BlueprintLoader
	fs              afero.Fs
	logger          *slog.Logger
	path            string
}

func (p *DefaultProjectLoader) Load() (Project, error) {
	p.logger.Info("Loading blueprint", "path", p.path)
	rbp, err := p.blueprintLoader.Load()
	if err != nil {
		p.logger.Error("Failed to load blueprint", "error", err, "path", p.path)
		return Project{}, fmt.Errorf("failed to load blueprint: %w", err)
	}

	bp, err := rbp.Decode()
	if err != nil {
		p.logger.Error("Failed to decode blueprint", "error", err)
		return Project{}, fmt.Errorf("failed to decode blueprint: %w", err)
	}

	efPath := filepath.Join(p.path, "Earthfile")
	exists, err := afero.Exists(p.fs, efPath)
	if err != nil {
		p.logger.Error("Failed to check for Earthfile", "error", err, "path", efPath)
		return Project{}, fmt.Errorf("failed to check for Earthfile: %w", err)
	}

	var ef *earthfile.Earthfile
	if exists {
		p.logger.Info("Parsing Earthfile", "path", efPath)
		eff, err := p.fs.Open(efPath)
		if err != nil {
			p.logger.Error("Failed to read Earthfile", "error", err, "path", efPath)
			return Project{}, fmt.Errorf("failed to read Earthfile: %w", err)
		}
		efs, err := earthfile.ParseEarthfile(context.Background(), eff)
		if err != nil {
			p.logger.Error("Failed to parse Earthfile", "error", err, "path", efPath)
			return Project{}, fmt.Errorf("failed to parse Earthfile: %w", err)
		}

		ef = &efs
	}

	return Project{
		Blueprint:    bp,
		Earthfile:    ef,
		Name:         bp.Project.Name,
		Path:         p.path,
		rawBlueprint: rbp,
	}, nil
}

// NewDefaultProjectLoader creates a new DefaultProjectLoader.
func NewDefaultProjectLoader(path string, logger *slog.Logger) DefaultProjectLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	bl := loader.NewDefaultBlueprintLoader(path, logger)
	return DefaultProjectLoader{
		blueprintLoader: &bl,
		fs:              afero.NewOsFs(),
		logger:          logger,
		path:            path,
	}
}

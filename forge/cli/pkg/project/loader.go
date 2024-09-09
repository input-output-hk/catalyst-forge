package project

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/loader"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthfile"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/git"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker"
	"github.com/spf13/afero"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks -out mocks/project.go . ProjectLoader

// ProjectLoader is an interface for loading projects.
type ProjectLoader interface {
	// Load loads the project.
	Load(projectPath string) (Project, error)
}

// DefaultProjectLoader is the default implementation of the ProjectLoader.
type DefaultProjectLoader struct {
	blueprintLoader loader.BlueprintLoader
	fs              afero.Fs
	logger          *slog.Logger
	repoLoader      git.RepoLoader
}

func (p *DefaultProjectLoader) Load(projectPath string) (Project, error) {
	p.logger.Info("Finding git root", "at", projectPath)
	w := walker.NewCustomReverseFSWalker(p.fs, p.logger)
	gitRoot, err := git.FindGitRoot(projectPath, &w)
	if err != nil {
		p.logger.Error("Failed to find git root", "error", err)
		return Project{}, fmt.Errorf("failed to find git root: %w", err)
	}

	rl := git.NewCustomDefaultRepoLoader(p.fs)
	repo, err := rl.Load(gitRoot)
	if err != nil {
		p.logger.Error("Failed to load repository", "error", err)
		return Project{}, fmt.Errorf("failed to load repository: %w", err)
	}

	p.logger.Info("Loading blueprint", "path", projectPath)
	rbp, err := p.blueprintLoader.Load(projectPath, gitRoot)
	if err != nil {
		p.logger.Error("Failed to load blueprint", "error", err, "path", projectPath)
		return Project{}, fmt.Errorf("failed to load blueprint: %w", err)
	}

	bp, err := rbp.Decode()
	if err != nil {
		p.logger.Error("Failed to decode blueprint", "error", err)
		return Project{}, fmt.Errorf("failed to decode blueprint: %w", err)
	}

	efPath := filepath.Join(projectPath, "Earthfile")
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
		Path:         projectPath,
		Repo:         repo,
		RepoRoot:     gitRoot,
		rawBlueprint: rbp,
	}, nil
}

// NewDefaultProjectLoader creates a new DefaultProjectLoader.
func NewDefaultProjectLoader(logger *slog.Logger) DefaultProjectLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	bl := loader.NewDefaultBlueprintLoader(logger)
	rl := git.NewDefaultRepoLoader()
	return DefaultProjectLoader{
		blueprintLoader: &bl,
		fs:              afero.NewOsFs(),
		logger:          logger,
		repoLoader:      &rl,
	}
}

package project

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/project/injector"
	"github.com/input-output-hk/catalyst-forge/lib/project/providers"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/earthfile"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
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
	blueprintLoader blueprint.BlueprintLoader
	ctx             *cue.Context
	fs              afero.Fs
	injectors       []injector.BlueprintInjector
	logger          *slog.Logger
	repoLoader      git.RepoLoader
	runtimes        []RuntimeData
}

func (p *DefaultProjectLoader) Load(projectPath string) (Project, error) {
	p.logger.Info("Finding git root", "projectPath", projectPath)
	w := walker.NewCustomReverseFSWalker(p.fs, p.logger)
	gitRoot, err := git.FindGitRoot(projectPath, &w)
	if err != nil {
		p.logger.Error("Failed to find git root", "error", err)
		return Project{}, fmt.Errorf("failed to find git root: %w", err)
	}

	p.logger.Info("Loading blueprint", "path", projectPath)
	rbp, err := p.blueprintLoader.Load(projectPath, gitRoot)
	if err != nil {
		p.logger.Error("Failed to load blueprint", "error", err, "path", projectPath)
		return Project{}, fmt.Errorf("failed to load blueprint: %w", err)
	}

	p.logger.Info("Loading repository", "path", gitRoot)
	rl := git.NewCustomDefaultRepoLoader(p.fs)
	repo, err := rl.Load(gitRoot)
	if err != nil {
		p.logger.Error("Failed to load repository", "error", err)
		return Project{}, fmt.Errorf("failed to load repository: %w", err)
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

	if !rbp.Get("project").Exists() {
		p.logger.Debug("No project config found in blueprint, assuming root config")
		bp, err := validateAndDecode(rbp)
		if err != nil {
			p.logger.Error("Failed loading blueprint", "error", err)
			return Project{}, fmt.Errorf("failed loading blueprint: %w", err)
		}

		return Project{
			Blueprint:    bp,
			Earthfile:    ef,
			Path:         projectPath,
			RawBlueprint: rbp,
			Repo:         repo,
			RepoRoot:     gitRoot,
			logger:       p.logger,
			ctx:          p.ctx,
		}, nil
	}

	var name string
	if err := rbp.DecodePath("project.name", &name); err != nil {
		return Project{}, fmt.Errorf("failed to get project name: %w", err)
	}

	p.logger.Info("Loading tag data")
	var tag *ProjectTag
	gitTag, err := git.GetTag(repo)
	if err != nil {
		p.logger.Warn("Failed to get git tag", "error", err)
	} else if gitTag != "" {
		t, err := ParseProjectTag(string(gitTag))
		if err != nil {
			p.logger.Warn("Failed to parse project tag", "error", err)
		} else if t.Project == name {
			tag = &t
		} else {
			p.logger.Debug("Git tag does not match project name", "tag", gitTag, "project", name)
		}
	} else {
		p.logger.Debug("No git tag found")
	}

	partialProject := Project{
		Earthfile:    ef,
		Name:         name,
		Path:         projectPath,
		RawBlueprint: rbp,
		Repo:         repo,
		RepoRoot:     gitRoot,
		Tag:          tag,
		ctx:          p.ctx,
		logger:       p.logger,
	}

	p.logger.Info("Gathering runtime data")
	runtimeData := make(map[string]cue.Value)
	for _, r := range p.runtimes {
		d := r.Load(&partialProject)

		for k, v := range d {
			runtimeData[k] = v
		}
	}

	p.logger.Info("Injecting blueprint")
	injs := append(p.injectors, injector.NewBlueprintRuntimeInjector(p.ctx, runtimeData, p.logger))
	for _, inj := range injs {
		rbp = inj.Inject(rbp)
	}

	if err := rbp.Validate(); err != nil {
		p.logger.Error("Failed to validate blueprint", "error", err)
		return Project{}, fmt.Errorf("failed to validate blueprint: %w", err)
	}

	bp, err := rbp.Decode()
	if err != nil {
		p.logger.Error("Failed to decode blueprint", "error", err)
		return Project{}, fmt.Errorf("failed to decode blueprint: %w", err)
	}

	return Project{
		Blueprint:    bp,
		ctx:          p.ctx,
		Earthfile:    ef,
		Name:         name,
		Path:         projectPath,
		RawBlueprint: rbp,
		Repo:         repo,
		RepoRoot:     gitRoot,
		logger:       p.logger,
		Tag:          tag,
	}, nil
}

// NewDefaultProjectLoader creates a new DefaultProjectLoader.
func NewDefaultProjectLoader(
	logger *slog.Logger,
) DefaultProjectLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	ctx := cuecontext.New()
	fs := afero.NewOsFs()
	bl := blueprint.NewDefaultBlueprintLoader(ctx, logger)
	rl := git.NewDefaultRepoLoader()
	store := secrets.NewDefaultSecretStore()
	ghp := providers.NewGithubProvider(fs, logger, &store)
	return DefaultProjectLoader{
		blueprintLoader: &bl,
		ctx:             ctx,
		fs:              fs,
		injectors: []injector.BlueprintInjector{
			injector.NewBlueprintEnvInjector(ctx, logger),
		},
		logger:     logger,
		repoLoader: &rl,
		runtimes: []RuntimeData{
			NewDeploymentRuntime(logger),
			NewGitRuntime(&ghp, logger),
		},
	}
}

// NewCustomProjectLoader creates a new DefaultProjectLoader with custom dependencies.
func NewCustomProjectLoader(
	ctx *cue.Context,
	fs afero.Fs,
	bl blueprint.BlueprintLoader,
	injectors []injector.BlueprintInjector,
	rl git.RepoLoader,
	runtimes []RuntimeData,
	logger *slog.Logger,
) DefaultProjectLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return DefaultProjectLoader{
		blueprintLoader: bl,
		ctx:             ctx,
		fs:              fs,
		injectors:       injectors,
		logger:          logger,
		repoLoader:      rl,
		runtimes:        runtimes,
	}
}

// validateAndDecode validates and decodes a raw blueprint.
func validateAndDecode(rbp blueprint.RawBlueprint) (schema.Blueprint, error) {
	if err := rbp.Validate(); err != nil {
		return schema.Blueprint{}, fmt.Errorf("failed to validate blueprint: %w", err)
	}

	bp, err := rbp.Decode()
	if err != nil {
		return schema.Blueprint{}, fmt.Errorf("failed to decode blueprint: %w", err)
	}

	return bp, nil
}

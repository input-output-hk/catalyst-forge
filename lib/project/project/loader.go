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
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/tools/earthfile"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
	r "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks -out mocks/project.go . ProjectLoader

// ProjectLoader is an interface for loading projects.
type ProjectLoader interface {
	// Load loads the project.
	Load(projectPath string) (Project, error)
}

type ProjectLoaderOption func(*DefaultProjectLoader)

// WithFs sets the filesystem for the project loader.
func WithFs(fs fs.Filesystem) ProjectLoaderOption {
	return func(p *DefaultProjectLoader) {
		p.fs = fs
	}
}

// WithInjectors sets the blueprint injectors for the project loader.
func WithInjectors(injectors []injector.BlueprintInjector) ProjectLoaderOption {
	return func(p *DefaultProjectLoader) {
		p.injectors = injectors
	}
}

// WithRuntimes sets the runtime data for the project loader.
func WithRuntimes(runtimes []RuntimeData) ProjectLoaderOption {
	return func(p *DefaultProjectLoader) {
		p.runtimes = runtimes
	}
}

// DefaultProjectLoader is the default implementation of the ProjectLoader.
type DefaultProjectLoader struct {
	blueprintLoader blueprint.BlueprintLoader
	ctx             *cue.Context
	fs              fs.Filesystem
	injectors       []injector.BlueprintInjector
	logger          *slog.Logger
	runtimes        []RuntimeData
	store           secrets.SecretStore
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
	repo, err := r.NewGitRepo(gitRoot, p.logger, r.WithFS(p.fs))
	if err != nil {
		p.logger.Error("Failed to create repository", "error", err)
	}

	if err := repo.Open(); err != nil {
		p.logger.Error("Failed to load repository", "error", err)
		return Project{}, fmt.Errorf("failed to load repository: %w", err)
	}

	efPath := filepath.Join(projectPath, "Earthfile")
	exists, err := p.fs.Exists(efPath)
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
			Repo:         &repo,
			RepoRoot:     gitRoot,
			SecretStore:  p.store,
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
	gitTag, err := git.GetTag(&repo)
	if err != nil {
		p.logger.Warn("Failed to get git tag", "error", err)
	} else if gitTag != "" {
		if !IsProjectTag(gitTag) {
			p.logger.Debug("Git tag is not a project tag", "tag", gitTag)
			tag = &ProjectTag{
				Full:    gitTag,
				Project: name,
				Version: gitTag,
			}
		} else {
			t, err := ParseProjectTag(gitTag)
			if err != nil {
				p.logger.Warn("Failed to parse project tag", "error", err)
			} else if t.Project == name {
				tag = &t
			} else {
				p.logger.Debug("Git tag does not match project name", "tag", gitTag, "project", name)
			}
		}
	} else {
		p.logger.Debug("No git tag found")
	}

	partialProject := Project{
		Earthfile:    ef,
		Name:         name,
		Path:         projectPath,
		RawBlueprint: rbp,
		Repo:         &repo,
		RepoRoot:     gitRoot,
		SecretStore:  p.store,
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
		Repo:         &repo,
		RepoRoot:     gitRoot,
		logger:       p.logger,
		SecretStore:  p.store,
		Tag:          tag,
	}, nil
}

// NewDefaultProjectLoader creates a new DefaultProjectLoader.
func NewDefaultProjectLoader(
	ctx *cue.Context,
	store secrets.SecretStore,
	logger *slog.Logger,
	opts ...ProjectLoaderOption,
) DefaultProjectLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	if ctx == nil {
		ctx = cuecontext.New()
	}

	fs := billy.NewBaseOsFS()
	l := DefaultProjectLoader{
		ctx: ctx,
		fs:  fs,
		injectors: []injector.BlueprintInjector{
			injector.NewBlueprintEnvInjector(ctx, logger),
			injector.NewBlueprintGlobalInjector(ctx, logger),
		},
		logger: logger,
		runtimes: []RuntimeData{
			NewDeploymentRuntime(logger),
			NewGitRuntime(logger),
		},
		store: store,
	}

	for _, o := range opts {
		o(&l)
	}

	bl := blueprint.NewCustomBlueprintLoader(ctx, l.fs, logger)
	l.blueprintLoader = &bl

	return l
}

// validateAndDecode validates and decodes a raw blueprint.
func validateAndDecode(rbp blueprint.RawBlueprint) (sb.Blueprint, error) {
	if err := rbp.Validate(); err != nil {
		return sb.Blueprint{}, fmt.Errorf("failed to validate blueprint: %w", err)
	}

	bp, err := rbp.Decode()
	if err != nil {
		return sb.Blueprint{}, fmt.Errorf("failed to decode blueprint: %w", err)
	}

	return bp, nil
}

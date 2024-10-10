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
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
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

	p.logger.Info("Loading blueprint", "path", projectPath)
	rbp, err := p.blueprintLoader.Load(projectPath, gitRoot)
	if err != nil {
		p.logger.Error("Failed to load blueprint", "error", err, "path", projectPath)
		return Project{}, fmt.Errorf("failed to load blueprint: %w", err)
	}

	p.logger.Info("Loading tag data")
	var tagConfig schema.Tagging
	var tagInfo *TagInfo
	if err := rbp.Get("global.ci.tagging").Decode(&tagConfig); err != nil {
		p.logger.Warn("Failed to load tag config", "error", err)
	} else {
		tagger := NewTagger(
			&Project{
				Blueprint: schema.Blueprint{
					Global: schema.Global{
						CI: schema.GlobalCI{
							Tagging: tagConfig,
						},
					},
				},
				ctx:       p.ctx,
				Earthfile: ef,
				Repo:      repo,
				RepoRoot:  gitRoot,
			},
			git.InCI(),
			true,
			p.logger,
		)

		t, err := tagger.GetTagInfo()
		if err != nil {
			p.logger.Error("Failed to get tag info", "error", err)
			tagInfo = nil
		} else {
			tagInfo = &t
		}
	}

	p.logger.Info("Gathering runtime data")
	runtimeData := make(map[string]cue.Value)
	for _, r := range p.runtimes {
		d := r.Load(&Project{
			ctx:          p.ctx,
			Earthfile:    ef,
			Repo:         repo,
			rawBlueprint: rbp,
			TagInfo:      tagInfo,
		})

		for k, v := range d {
			runtimeData[k] = v
		}
	}

	p.logger.Info("Injecting blueprint")
	p.injectors = append(p.injectors, injector.NewBlueprintRuntimeInjector(p.ctx, runtimeData, p.logger))
	for _, inj := range p.injectors {
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
		Name:         bp.Project.Name,
		Path:         projectPath,
		Repo:         repo,
		RepoRoot:     gitRoot,
		logger:       p.logger,
		rawBlueprint: rbp,
		TagInfo:      tagInfo,
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
	bl := blueprint.NewDefaultBlueprintLoader(ctx, logger)
	rl := git.NewDefaultRepoLoader()
	return DefaultProjectLoader{
		blueprintLoader: &bl,
		ctx:             ctx,
		fs:              afero.NewOsFs(),
		injectors: []injector.BlueprintInjector{
			injector.NewBlueprintEnvInjector(ctx, logger),
		},
		logger:     logger,
		repoLoader: &rl,
		runtimes:   GetDefaultRuntimes(logger),
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

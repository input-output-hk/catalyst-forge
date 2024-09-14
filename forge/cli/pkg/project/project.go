package project

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	gg "github.com/go-git/go-git/v5"
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/blueprint"
	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthfile"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets"
)

type TagInfo struct {
	Generated string `json:"generated"`
	Git       string `json:"git"`
}

// Project represents a project
type Project struct {
	Blueprint    schema.Blueprint
	CI           bool
	Earthfile    *earthfile.Earthfile
	Local        bool
	Name         string
	Path         string
	Repo         *gg.Repository
	RepoRoot     string
	Tags         TagInfo
	logger       *slog.Logger
	rawBlueprint blueprint.RawBlueprint
}

// GetRelativePath returns the relative path of the project from the repo root.
func (p *Project) GetRelativePath() (string, error) {
	var projectPath, repoRoot string
	var err error

	if !filepath.IsAbs(p.Path) {
		projectPath, err = filepath.Abs(p.Path)
		if err != nil {
			return "", fmt.Errorf("failed to get project path: %w", err)
		}
	} else {
		projectPath = p.Path
	}

	if !filepath.IsAbs(p.RepoRoot) {
		repoRoot, err = filepath.Abs(p.RepoRoot)
		if err != nil {
			return "", fmt.Errorf("failed to get repo root: %w", err)
		}
	} else {
		repoRoot = p.RepoRoot
	}

	if !strings.HasPrefix(projectPath, repoRoot) {
		return "", fmt.Errorf("project path is not a subdirectory of the repo root")
	}

	relPath, err := filepath.Rel(repoRoot, projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	return relPath, nil
}

// Raw returns the raw blueprint.
func (p *Project) Raw() blueprint.RawBlueprint {
	return p.rawBlueprint
}

// RunTarget runs the given Earthly target.
func (p *Project) RunTarget(
	target string,
	exec executor.Executor,
	store secrets.SecretStore,
	opts ...earthly.EarthlyExecutorOption,
) (map[string]earthly.EarthlyExecutionResult, error) {
	return earthly.NewEarthlyExecutor(
		p.Path,
		target,
		exec,
		store,
		p.logger,
		append(p.generateOpts(target), opts...)...,
	).Run()
}

// generateOpts generates the options for the Earthly executor.
func (p *Project) generateOpts(target string) []earthly.EarthlyExecutorOption {
	var opts []earthly.EarthlyExecutorOption

	if _, ok := p.Blueprint.Project.CI.Targets[target]; ok {
		targetConfig := p.Blueprint.Project.CI.Targets[target]

		if len(targetConfig.Args) > 0 {
			var args []string
			for k, v := range targetConfig.Args {
				args = append(args, fmt.Sprintf("--%s", k), v)
			}

			opts = append(opts, earthly.WithTargetArgs(args...))
		}

		// We only run multiple platforms in CI mode to avoid issues with local builds.
		if targetConfig.Platforms != nil && p.CI {
			opts = append(opts, earthly.WithPlatforms(targetConfig.Platforms...))
		}

		if targetConfig.Privileged != nil && *targetConfig.Privileged {
			opts = append(opts, earthly.WithPrivileged())
		}

		if targetConfig.Retries != nil {
			opts = append(opts, earthly.WithRetries(*targetConfig.Retries))
		}

		if len(targetConfig.Secrets) > 0 {
			opts = append(opts, earthly.WithSecrets(targetConfig.Secrets))
		}
	}

	if p.Blueprint.Global.CI.Providers.Earthly.Satellite != nil && !p.Local {
		opts = append(opts, earthly.WithSatellite(*p.Blueprint.Global.CI.Providers.Earthly.Satellite))
	}

	if len(p.Blueprint.Global.CI.Secrets) > 0 {
		opts = append(opts, earthly.WithSecrets(p.Blueprint.Global.CI.Secrets))
	}

	return opts
}

package providers

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
	"github.com/spf13/afero"
)

type DocsReleaserConfig struct {
	Branch     string                     `json:"branch"`
	Branches   DocsReleaserBranchesConfig `json:"branches"`
	TargetPath string                     `json:"targetPath"`
	Token      schema.Secret              `json:"token"`
}

type DocsReleaserBranchesConfig struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
}

type DocsReleaser struct {
	config      DocsReleaserConfig
	force       bool
	fs          afero.Fs
	handler     events.EventHandler
	logger      *slog.Logger
	project     project.Project
	release     schema.Release
	releaseName string
	runner      run.ProjectRunner
	token       string
	workdir     string
}

func (r *DocsReleaser) Release() error {
	r.logger.Info("Running release target", "project", r.project.Name, "target", r.release.Target, "dir", r.workdir)
	if err := r.run(r.workdir); err != nil {
		return fmt.Errorf("failed to run release target: %w", err)
	}

	if err := r.validateArtifacts(r.workdir); err != nil {
		return fmt.Errorf("failed to validate artifacts: %w", err)
	}

	if !r.handler.Firing(&r.project, r.project.GetReleaseEvents(r.releaseName)) && !r.force {
		r.logger.Info("No release event is firing, skipping release")
		return nil
	}

	r.logger.Debug("Getting current branch")
	curBranch, err := git.GetCurrentBranch(r.project.Repo)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	r.logger.Info("Checking out branch", "branch", r.config.Branch, "current", curBranch)
	if err := git.CheckoutBranch(
		r.project.Repo,
		r.config.Branch,
		git.GitCheckoutCreate(),
		git.GitCheckoutForceClean(),
	); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	if err := r.clean(); err != nil {
		return fmt.Errorf("failed to clean target path: %w", err)
	}

	// r.logger.Debug("Restoring branch", "branch", curBranch)
	// if err := git.CheckoutBranch(r.project.Repo, curBranch, git.GitCheckoutCreate()); err != nil {
	// 	return fmt.Errorf("failed to checkout branch: %w", err)
	// }

	return nil
}

func (r *DocsReleaser) clean() error {
	targetPath := filepath.Join(r.project.RepoRoot, r.config.TargetPath)

	_, err := os.Stat(targetPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to get file info: %w", err)
		} else {
			r.logger.Debug("Target path does not exist, skipping clean", "path", targetPath)
			return nil
		}
	}

	var branchPath string
	if r.config.Branches.Enabled {
		branchPath = filepath.Join(targetPath, r.config.Branches.Path)
		r.logger.Debug("Cleaning branch path", "path", branchPath)
	}

	r.logger.Info("Cleaning target path", "path", targetPath)
	err = afero.Walk(r.fs, targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk path: %w", err)
		}

		if info.IsDir() {
			if path == branchPath {
				r.logger.Debug("Skipping branch path", "path", path)
				return filepath.SkipDir
			} else if path == filepath.Join(r.project.RepoRoot, ".git") {
				r.logger.Debug("Skipping git path", "path", path)
				return filepath.SkipDir
			}
		}

		r.logger.Debug("Removing file", "path", path)
		// if err := r.fs.Remove(path); err != nil {
		// 	return fmt.Errorf("failed to remove file: %w", err)
		// }

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to clean target path: %w", err)
	}

	return nil
}

// run runs the release target.
func (r *DocsReleaser) run(path string) error {
	return r.runner.RunTarget(
		r.release.Target,
		earthly.WithArtifact(path),
	)
}

// validateArtifacts checks if the artifacts exist.
func (r *DocsReleaser) validateArtifacts(path string) error {
	r.logger.Info("Validating artifacts")
	path = filepath.Join(path, earthly.GetBuildPlatform())
	exists, err := afero.DirExists(r.fs, path)
	if err != nil {
		return fmt.Errorf("failed to check if output folder exists: %w", err)
	} else if !exists {
		return fmt.Errorf("unable to find output folder for platform: %s", path)
	}

	children, err := afero.ReadDir(r.fs, path)
	if err != nil {
		return fmt.Errorf("failed to read output folder: %w", err)
	}

	if len(children) == 0 {
		return fmt.Errorf("no artifacts found")
	}

	return nil
}

func NewDocsReleaser(
	ctx run.RunContext,
	project project.Project,
	name string,
	force bool,
) (*DocsReleaser, error) {
	release, ok := project.Blueprint.Project.Release[name]
	if !ok {
		return nil, fmt.Errorf("unknown release: %s", name)
	}

	var config DocsReleaserConfig
	if err := parseConfig(&project, name, &config); err != nil {
		return nil, fmt.Errorf("failed to parse release config: %w", err)
	}

	if config.Branch == "" {
		return nil, fmt.Errorf("must specify a branch to checkout")
	} else if config.TargetPath == "" {
		return nil, fmt.Errorf("must specify a target path")
	} else if filepath.IsAbs(config.TargetPath) {
		return nil, fmt.Errorf("target path must be relative")
	} else if config.Branches.Enabled && config.Branches.Path == "" {
		return nil, fmt.Errorf("must specify a branch path if branches are enabled")
	} else if config.Branches.Enabled && filepath.IsAbs(config.Branches.Path) {
		return nil, fmt.Errorf("branch path must be relative")
	}

	token, err := secrets.GetSecret(&config.Token, &ctx.SecretStore, ctx.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	fs := afero.NewOsFs()
	workdir, err := afero.TempDir(fs, "", "catalyst-forge-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	handler := events.NewDefaultEventHandler(ctx.Logger)
	runner := run.NewDefaultProjectRunner(ctx, &project)
	return &DocsReleaser{
		config:      config,
		force:       force,
		fs:          fs,
		handler:     &handler,
		logger:      ctx.Logger,
		project:     project,
		release:     release,
		releaseName: name,
		runner:      &runner,
		token:       token,
		workdir:     workdir,
	}, nil
}

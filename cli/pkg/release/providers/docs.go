package providers

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
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

	if err := r.checkoutBranch(); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	return nil
}

func (r *DocsReleaser) checkoutBranch() error {
	if r.config.Branch == "" {
		return fmt.Errorf("must specify a branch to checkout")
	}

	r.logger.Info("Checking out branch", "branch", r.config.Branch)
	wt, err := r.project.Repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get git worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return fmt.Errorf("failed to get git worktree status: %w", err)
	}

	if !status.IsClean() {
		return fmt.Errorf("refusing to proceed due to dirty git worktree")
	}

	if err := wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(r.config.Branch),
		Create: true,
	}); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
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

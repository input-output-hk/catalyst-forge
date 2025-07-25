package providers

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	gh "github.com/input-output-hk/catalyst-forge/lib/providers/github"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/archive"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

type GithubReleaserConfig struct {
	Prefix string `json:"prefix"`
	Name   string `json:"name"`
}

type GithubReleaser struct {
	client      gh.GithubClient
	config      GithubReleaserConfig
	force       bool
	fs          fs.Filesystem
	handler     events.EventHandler
	logger      *slog.Logger
	project     project.Project
	release     sp.Release
	releaseName string
	runner      earthly.ProjectRunner
	workdir     string
}

func (r *GithubReleaser) Release() error {
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

	if r.project.Tag == nil {
		return fmt.Errorf("cannot create a release without a git tag")
	}

	var assets []string
	for _, platform := range r.getPlatforms() {
		filename := fmt.Sprintf("%s-%s.tar.gz", r.config.Prefix, strings.Replace(platform, "/", "-", -1))
		destpath := filepath.Join(r.workdir, filename)
		srcpath := filepath.Join(r.workdir, platform)

		r.logger.Info("Creating archive", "srcpath", srcpath, "filename", filename)
		if err := archive.ArchiveAndCompress(r.fs, srcpath, destpath); err != nil {
			return fmt.Errorf("failed to archive and compress: %w", err)
		}

		assets = append(assets, filename)
	}

	release, err := r.client.GetReleaseByTag(r.project.Tag.Full)
	if errors.Is(err, gh.ErrReleaseNotFound) {
		r.logger.Info("Creating release", "name", r.config.Name)
		release, err = r.client.CreateRelease(&github.RepositoryRelease{
			Name:    &r.config.Name,
			TagName: &r.project.Tag.Full,
			Draft:   github.Bool(false),
		})

		if err != nil {
			return fmt.Errorf("failed to create release: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get release: %w", err)
	} else {
		r.logger.Info("Release already exists", "name", r.config.Name)
	}

	for _, asset := range assets {
		if assetExists(*release, asset) {
			r.logger.Warn("Asset already exists", "asset", asset)
			continue
		}

		r.logger.Info("Uploading asset", "asset", asset)
		if err := r.client.UploadReleaseAsset(*release.ID, filepath.Join(r.workdir, asset)); err != nil {
			return fmt.Errorf("failed to upload asset: %w", err)
		}
	}

	return nil
}

// getPlatforms returns the current platforms.
func (r *GithubReleaser) getPlatforms() []string {
	var platforms []string
	platforms = getPlatforms(&r.project, r.release.Target)

	if platforms == nil {
		platforms = []string{earthly.GetBuildPlatform()}
	}

	return platforms
}

// run runs the release target.
func (r *GithubReleaser) run(path string) error {
	return r.runner.RunTarget(
		r.release.Target,
		earthly.WithArtifact(path),
	)
}

func (r *GithubReleaser) validateArtifacts(path string) error {
	for _, platform := range r.getPlatforms() {
		r.logger.Info("Validating artifacts", "platform", platform)
		path := filepath.Join(path, platform)
		exists, err := r.fs.Exists(path)
		if err != nil {
			return fmt.Errorf("failed to check if output folder exists: %w", err)
		} else if !exists {
			return fmt.Errorf("unable to find output folder for platform: %s", path)
		}

		children, err := r.fs.ReadDir(path)
		if err != nil {
			return fmt.Errorf("failed to read output folder: %w", err)
		}

		if len(children) == 0 {
			return fmt.Errorf("no artifacts found for platform: %s", platform)
		}
	}

	return nil
}

// assetExists checks if an asset exists in a release.
func assetExists(release github.RepositoryRelease, name string) bool {
	for _, asset := range release.Assets {
		if *asset.Name == name {
			return true
		}
	}

	return false
}

func NewGithubReleaser(
	ctx run.RunContext,
	project project.Project,
	name string,
	force bool,
) (*GithubReleaser, error) {
	release, ok := project.Blueprint.Project.Release[name]
	if !ok {
		return nil, fmt.Errorf("unknown release: %s", name)
	}

	var config GithubReleaserConfig
	if err := parseConfig(&project, name, &config); err != nil {
		return nil, fmt.Errorf("failed to parse release config: %w", err)
	}

	fs := billy.NewBaseOsFS()
	workdir, err := fs.TempDir("", "catalyst-forge-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	owner := strings.Split(project.Blueprint.Global.Repo.Name, "/")[0]
	repo := strings.Split(project.Blueprint.Global.Repo.Name, "/")[1]
	client, err := gh.NewDefaultGithubClient(
		owner,
		repo,
		gh.WithCredsOrEnv(project.Blueprint.Global.Ci.Providers.Github.Credentials),
		gh.WithLogger(ctx.Logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create github client: %w", err)
	}

	handler := events.NewDefaultEventHandler(ctx.Logger)
	runner := earthly.NewDefaultProjectRunner(ctx, &project)
	return &GithubReleaser{
		config:      config,
		client:      client,
		force:       force,
		fs:          fs,
		handler:     &handler,
		logger:      ctx.Logger,
		project:     project,
		release:     release,
		releaseName: name,
		runner:      &runner,
		workdir:     workdir,
	}, nil
}

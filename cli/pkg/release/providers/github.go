package providers

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/release/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/archive"
	"github.com/spf13/afero"
)

type GithubClient interface {
	RepositoriesGetReleaseByTag(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, *github.Response, error)
}

type GithubReleaserConfig struct {
	Token schema.Secret `json:"token"`
}

type GithubReleaser struct {
	client  *github.Client
	force   bool
	fs      afero.Fs
	handler events.ReleaseEventHandler
	logger  *slog.Logger
	project project.Project
	release schema.Release
	runner  run.ProjectRunner
	workdir string
}

func (r *GithubReleaser) Release() error {
	r.logger.Info("Running release target", "project", r.project.Name, "target", r.release.Target, "dir", r.workdir)
	if err := r.run(r.workdir); err != nil {
		return fmt.Errorf("failed to run release target: %w", err)
	}

	if err := r.validateArtifacts(r.workdir); err != nil {
		return fmt.Errorf("failed to validate artifacts: %w", err)
	}

	if !r.handler.Firing(r.release.On) && !r.force {
		r.logger.Info("No release event is firing, skipping release")
		return nil
	}

	var assets []string
	for _, platform := range r.getPlatforms() {
		filename := fmt.Sprintf("%s-%s.tar.gz", r.project.Name, strings.Replace(platform, "/", "-", -1))
		destpath := filepath.Join(r.workdir, filename)
		srcpath := filepath.Join(r.workdir, platform)

		r.logger.Info("Creating archive", "srcpath", srcpath, "filename", filename)
		if err := archive.ArchiveAndCompress(r.fs, srcpath, destpath); err != nil {
			return fmt.Errorf("failed to archive and compress: %w", err)
		}

		assets = append(assets, filename)
	}

	parts := strings.Split(r.project.Blueprint.Global.Repo.Name, "/")
	owner, repo := parts[0], parts[1]

	releaseName := string(r.project.TagInfo.Git)
	ctx := context.Background()

	release, resp, err := r.client.Repositories.GetReleaseByTag(ctx, owner, repo, releaseName)
	if resp.StatusCode == 404 {
		r.logger.Info("Creating release", "name", releaseName)
		release, _, err = r.client.Repositories.CreateRelease(ctx, owner, repo, &github.RepositoryRelease{
			Name:    &releaseName,
			TagName: &releaseName,
			Draft:   github.Bool(false),
		})

		if err != nil {
			return fmt.Errorf("failed to create release: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get release: %w", err)
	} else {
		r.logger.Info("Release already exists", "name", releaseName)
	}

	for _, asset := range assets {
		if assetExists(*release, asset) {
			r.logger.Warn("Asset already exists", "asset", asset)
			continue
		}

		r.logger.Info("Uploading asset", "asset", asset)
		f, err := r.fs.Open(filepath.Join(r.workdir, asset))
		if err != nil {
			return fmt.Errorf("failed to open asset: %w", err)
		}

		stat, err := f.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat asset: %w", err)
		}

		url := fmt.Sprintf("repos/%s/%s/releases/%d/assets?name=%s", owner, repo, *release.ID, asset)
		req, err := r.client.NewUploadRequest(url, f, stat.Size(), "application/gzip")
		if err != nil {
			return fmt.Errorf("failed to create upload request: %w", err)
		}

		_, err = r.client.Do(ctx, req, nil)
		if err != nil {
			return fmt.Errorf("failed to upload asset: %w", err)
		}

		f.Close()
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

	token, err := secrets.GetSecret(&config.Token, &ctx.SecretStore, ctx.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	fs := afero.NewOsFs()
	workdir, err := afero.TempDir(fs, "", "catalyst-forge-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	client := github.NewClient(nil).WithAuthToken(token)
	handler := events.NewDefaultReleaseEventHandler(&project, ctx.Logger)
	runner := run.NewDefaultProjectRunner(ctx, &project)
	return &GithubReleaser{
		client:  client,
		force:   force,
		fs:      fs,
		handler: &handler,
		logger:  ctx.Logger,
		project: project,
		release: release,
		runner:  &runner,
		workdir: workdir,
	}, nil
}

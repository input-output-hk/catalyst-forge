package providers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
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
}

func (r *GithubReleaser) Release() error {
	// tmp, err := afero.TempDir(r.fs, "", "catalyst-forge-")
	// if err != nil {
	// 	return fmt.Errorf("failed to create temporary directory: %w", err)
	// }

	// r.logger.Info("Running release target", "project", r.project.Name, "target", r.release.Target, "dir", tmp)
	// if err := r.run(tmp); err != nil {
	// 	return fmt.Errorf("failed to run release target: %w", err)
	// }

	tmp := "/tmp/catalyst-forge-802916104"
	if err := r.validateArtifacts(tmp); err != nil {
		return fmt.Errorf("failed to validate artifacts: %w", err)
	}

	var assets []string
	for _, platform := range r.getPlatforms() {
		filename := fmt.Sprintf("%s-%s.tar.gz", r.project.Name, strings.Replace(platform, "/", "-", -1))
		destpath := filepath.Join(tmp, filename)
		srcpath := filepath.Join(tmp, platform)

		r.logger.Info("Creating archive", "srcpath", srcpath, "filename", filename)
		if err := archive.ArchiveAndCompress(r.fs, srcpath, destpath); err != nil {
			return fmt.Errorf("failed to archive and compress: %w", err)
		}

		assets = append(assets, filename)
	}

	var owner, repo string
	if gr, ok := os.LookupEnv("GITHUB_REPOSITORY"); ok {
		parts := strings.Split(gr, "/")
		owner, repo = parts[0], parts[1]
	} else {
		parts := strings.Split(r.project.Blueprint.Global.Repo.Name, "/")
		owner, repo = parts[0], parts[1]
	}

	//releaseName := string(r.project.TagInfo.Git)
	releaseName := "testing"
	ctx := context.Background()
	_, _, err := r.client.Repositories.CreateRelease(ctx, owner, repo, &github.RepositoryRelease{
		Name:    &releaseName,
		TagName: &releaseName,
	})
	if err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	return nil
}

// getPlatforms returns the current platforms.
func (r *GithubReleaser) getPlatforms() []string {
	var platforms []string
	platforms = getPlatforms(&r.project, r.release.Target)

	if platforms == nil {
		platforms = []string{fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)}
	}

	return platforms
}

// run runs the release target.
func (r *GithubReleaser) run(path string) error {
	_, err := r.runner.RunTarget(
		r.release.Target,
		earthly.WithArtifact(path),
	)

	return err
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
	if err := project.Raw().DecodePath(fmt.Sprintf("project.release.%s.config", name), &config); err != nil {
		return nil, fmt.Errorf("failed to decode release config: %w", err)
	}

	token, err := secrets.GetSecret(&config.Token, &ctx.SecretStore, ctx.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	client := github.NewClient(nil).WithAuthToken(token)
	handler := events.NewDefaultReleaseEventHandler(&project, ctx.Logger)
	runner := run.NewDefaultProjectRunner(ctx, &project)
	return &GithubReleaser{
		client:  client,
		force:   force,
		fs:      afero.NewOsFs(),
		handler: &handler,
		logger:  ctx.Logger,
		project: project,
		release: release,
		runner:  &runner,
	}, nil
}

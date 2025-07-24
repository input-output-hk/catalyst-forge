package providers

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/aws"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/providers"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/github"
)

// DocsReleaserConfig is the configuration for the docs release.
type DocsReleaserConfig struct {
	Name string `json:"name"`
}

// DocsReleaser is a provider that releases the docs for a project.
type DocsReleaser struct {
	config      DocsReleaserConfig
	force       bool
	fs          fs.Filesystem
	handler     events.EventHandler
	logger      *slog.Logger
	project     *project.Project
	release     sp.Release
	releaseName string
	runner      earthly.ProjectRunner
	s3          aws.S3Client
	workdir     string
}

// Release runs the docs release.
func (r *DocsReleaser) Release() error {
	// r.logger.Info("Running docs release target", "project", r.project.Name, "target", r.release.Target, "dir", r.workdir)
	// if err := r.run(r.workdir); err != nil {
	// 	return fmt.Errorf("failed to run docs release target: %w", err)
	// }

	// if err := r.validateArtifacts(r.workdir); err != nil {
	// 	return fmt.Errorf("failed to validate artifacts: %w", err)
	// }

	if !r.handler.Firing(r.project, r.project.GetReleaseEvents(r.releaseName)) && !r.force {
		r.logger.Info("No release event is firing, skipping release")
		return nil
	}

	env := github.NewGithubEnv(r.logger)
	if env.IsPR() {
		pr := env.GetPRNumber()
		if pr == 0 {
			return fmt.Errorf("failed to get PR number")
		}

		client, err := providers.NewGithubClient(r.project, r.logger)
		if err != nil {
			return fmt.Errorf("failed to create github client: %w", err)
		}

		prClient := github.NewPRClient(client, r.logger)
		owner := strings.Split(r.project.Blueprint.Global.Repo.Name, "/")[0]
		repo := strings.Split(r.project.Blueprint.Global.Repo.Name, "/")[1]

		comments, err := prClient.ListComments(owner, repo, pr)
		if err != nil {
			return fmt.Errorf("failed to list comments: %w", err)
		}

		for _, comment := range comments {
			r.logger.Info("Comment", "author", comment.Author, "body", comment.Body)
		}

		// if err := prClient.PostComment(owner, repo, pr, "Hello, world!"); err != nil {
		// 	return fmt.Errorf("failed to post comment to PR: %w", err)
		// }
	}

	// if r.project.Blueprint.Global.Ci == nil || r.project.Blueprint.Global.Ci.Release == nil || r.project.Blueprint.Global.Ci.Release.Docs == nil {
	// 	return fmt.Errorf("global docs release configuration not found")
	// }

	// docsConfig := r.project.Blueprint.Global.Ci.Release.Docs
	// if docsConfig.Bucket == "" {
	// 	return fmt.Errorf("no S3 bucket specified in global docs configuration")
	// }

	// s3Path, err := r.generatePath()
	// if err != nil {
	// 	return fmt.Errorf("failed to generate S3 path: %w", err)
	// }

	// r.logger.Info("Cleaning existing docs from S3", "bucket", docsConfig.Bucket, "path", s3Path)
	// if err := r.s3.DeleteDirectory(docsConfig.Bucket, s3Path); err != nil {
	// 	return fmt.Errorf("failed to clean existing docs from S3: %w", err)
	// }

	// finalPath := filepath.Join(r.workdir, earthly.GetBuildPlatform())
	// r.logger.Info("Uploading docs to S3", "bucket", docsConfig.Bucket, "path", s3Path)
	// if err := r.s3.UploadDirectory(docsConfig.Bucket, s3Path, finalPath, r.fs); err != nil {
	// 	return fmt.Errorf("failed to upload docs to S3: %w", err)
	// }

	r.logger.Info("Docs release complete")
	return nil
}

// generatePath generates the S3 path for the docs.
func (r *DocsReleaser) generatePath() (string, error) {
	projectName := r.config.Name
	if projectName == "" {
		projectName = r.project.Name
	}

	docsConfig := r.project.Blueprint.Global.Ci.Release.Docs
	if docsConfig.Bucket == "" {
		return "", fmt.Errorf("no S3 bucket specified in global docs configuration")
	}

	s3Path := projectName
	if docsConfig.Path != "" {
		s3Path = docsConfig.Path + "/" + projectName
	}

	branch, err := git.GetBranch(r.project.Repo)
	if err != nil {
		return "", fmt.Errorf("failed to get branch: %w", err)
	}

	if branch == r.project.Blueprint.Global.Repo.DefaultBranch {
		return s3Path, nil
	}

	return s3Path + "/" + branch, nil
}

// run runs the docs release target.
func (r *DocsReleaser) run(path string) error {
	return r.runner.RunTarget(
		r.release.Target,
		earthly.WithArtifact(path),
	)
}

// validateArtifacts validates that the expected artifacts exist.
func (r *DocsReleaser) validateArtifacts(path string) error {
	r.logger.Info("Validating docs artifacts", "path", path)
	exists, err := r.fs.Exists(path)
	if err != nil {
		return fmt.Errorf("failed to check if output folder exists: %w", err)
	} else if !exists {
		return fmt.Errorf("unable to find output folder: %s", path)
	}

	children, err := r.fs.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read output folder: %w", err)
	}

	if len(children) == 0 {
		return fmt.Errorf("no docs artifacts found")
	}

	return nil
}

// NewDocsReleaser creates a new docs release provider.
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

	fs := billy.NewBaseOsFS()
	workdir, err := fs.TempDir("", "catalyst-forge-docs-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	s3, err := aws.NewS3Client(ctx.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	handler := events.NewDefaultEventHandler(ctx.Logger)
	runner := earthly.NewDefaultProjectRunner(ctx, &project)
	return &DocsReleaser{
		config:      config,
		force:       force,
		fs:          fs,
		handler:     &handler,
		logger:      ctx.Logger,
		project:     &project,
		release:     release,
		releaseName: name,
		runner:      &runner,
		s3:          s3,
		workdir:     workdir,
	}, nil
}

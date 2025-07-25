package providers

import (
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthly"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/events"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/aws"
	"github.com/input-output-hk/catalyst-forge/lib/providers/git"
	"github.com/input-output-hk/catalyst-forge/lib/providers/github"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

const (
	docsCommentPrefix = "<!-- forge:v1:docs-preview -->"
	bodyTemplate      = `
%s
## 📚 Docs Preview

The docs for this PR can be previewed at the following URL:

%s
`
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
	ghClient    github.GithubClient
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
	r.logger.Info("Running docs release target", "project", r.project.Name, "target", r.release.Target, "dir", r.workdir)
	if err := r.run(r.workdir); err != nil {
		return fmt.Errorf("failed to run docs release target: %w", err)
	}

	if err := r.validateArtifacts(r.workdir); err != nil {
		return fmt.Errorf("failed to validate artifacts: %w", err)
	}

	if !r.handler.Firing(r.project, r.project.GetReleaseEvents(r.releaseName)) && !r.force {
		r.logger.Info("No release event is firing, skipping release")
		return nil
	}

	if r.project.Blueprint.Global.Ci == nil || r.project.Blueprint.Global.Ci.Release == nil || r.project.Blueprint.Global.Ci.Release.Docs == nil {
		return fmt.Errorf("global docs release configuration not found")
	}

	projectName := r.config.Name
	if projectName == "" {
		projectName = r.project.Name
	}

	docsConfig := r.project.Blueprint.Global.Ci.Release.Docs
	if docsConfig.Bucket == "" {
		return fmt.Errorf("no S3 bucket specified in global docs configuration")
	}

	s3Path, err := r.generatePath(projectName)
	if err != nil {
		return fmt.Errorf("failed to generate S3 path: %w", err)
	}

	r.logger.Info("Cleaning existing docs from S3", "bucket", docsConfig.Bucket, "path", s3Path)
	if err := r.s3.DeleteDirectory(docsConfig.Bucket, s3Path, nil); err != nil {
		return fmt.Errorf("failed to clean existing docs from S3: %w", err)
	}

	finalPath := filepath.Join(r.workdir, earthly.GetBuildPlatform())
	r.logger.Info("Uploading docs to S3", "bucket", docsConfig.Bucket, "path", s3Path)
	if err := r.s3.UploadDirectory(docsConfig.Bucket, s3Path, finalPath, r.fs); err != nil {
		return fmt.Errorf("failed to upload docs to S3: %w", err)
	}

	if github.InCI() {
		r.logger.Info("Posting comment", "url", docsConfig.Url, "project", projectName)
		url := r.project.Blueprint.Global.Ci.Release.Docs.Url
		if err := r.postComment(url, projectName); err != nil {
			return fmt.Errorf("failed to post comment: %w", err)
		}

		isDefault, err := r.isDefaultBranch()
		if err != nil {
			return fmt.Errorf("failed to check if branch is default: %w", err)
		}

		if isDefault {
			r.logger.Info("Cleaning up branches from S3", "bucket", docsConfig.Bucket, "path", filepath.Dir(s3Path))
			if err := r.cleanupBranches(docsConfig.Bucket, filepath.Dir(s3Path)); err != nil {
				return fmt.Errorf("failed to cleanup branches: %w", err)
			}
		}
	}

	r.logger.Info("Docs release complete")
	return nil
}

// cleanupBranches deletes branches from S3 that are no longer present in GitHub.
func (r *DocsReleaser) cleanupBranches(bucket, path string) error {
	branches, err := r.ghClient.ListBranches()
	if err != nil {
		return fmt.Errorf("failed to list GitHub branches: %w", err)
	}

	var branchNames []string
	for _, branch := range branches {
		branchNames = append(branchNames, branch.Name)
	}
	r.logger.Info("Repo branches", "branches", branchNames)

	children, err := r.s3.ListImmediateChildren(bucket, path)
	if err != nil {
		return fmt.Errorf("failed to list immediate children: %w", err)
	}
	r.logger.Info("Docs branches", "branches", children)

	for _, child := range children {
		if !slices.Contains(branchNames, child) {
			r.logger.Info("Deleting branch", "branch", child)
			if err := r.s3.DeleteDirectory(bucket, filepath.Join(path, child), nil); err != nil {
				return fmt.Errorf("failed to delete branch: %w", err)
			}
		}
	}

	return nil
}

// generatePath generates the S3 path for the docs.
func (r *DocsReleaser) generatePath(projectName string) (string, error) {
	docsConfig := r.project.Blueprint.Global.Ci.Release.Docs
	branch, err := git.GetBranch(r.ghClient, r.project.Repo)
	if err != nil {
		return "", fmt.Errorf("failed to get branch: %w", err)
	}

	var s3Path string
	if docsConfig.Path != "" {
		s3Path = filepath.Join(docsConfig.Path, projectName, branch)
	} else {
		s3Path = filepath.Join(projectName, branch)
	}

	return s3Path, nil
}

// isDefaultBranch returns true if the current branch is the default branch.
func (r *DocsReleaser) isDefaultBranch() (bool, error) {
	branch, err := git.GetBranch(r.ghClient, r.project.Repo)
	if err != nil {
		return false, fmt.Errorf("failed to get branch: %w", err)
	}

	return branch == r.project.Blueprint.Global.Repo.DefaultBranch, nil
}

// postComment posts a comment to the PR.
func (r *DocsReleaser) postComment(baseURL, name string) error {
	if r.ghClient.Env().IsPR() {
		pr := r.ghClient.Env().GetPRNumber()
		if pr == 0 {
			r.logger.Warn("No PR number found, skipping comment")
			return nil
		}

		comments, err := r.ghClient.ListPullRequestComments(pr)
		if err != nil {
			return fmt.Errorf("failed to list comments: %w", err)
		}

		for _, comment := range comments {
			if comment.Author == "github-actions[bot]" && strings.Contains(comment.Body, docsCommentPrefix) {
				r.logger.Info("Found existing comment, skipping	")
				return nil
			}
		}

		branch, err := git.GetBranch(r.ghClient, r.project.Repo)
		if err != nil {
			return fmt.Errorf("failed to get branch: %w", err)
		}

		docURL, err := url.JoinPath(baseURL, name, branch)
		if err != nil {
			return fmt.Errorf("failed to join URL path: %w", err)
		}

		body := fmt.Sprintf(bodyTemplate, docsCommentPrefix, docURL)
		if err := r.ghClient.PostPullRequestComment(pr, body); err != nil {
			return fmt.Errorf("failed to post comment to PR: %w", err)
		}
	} else {
		r.logger.Info("No PR found, skipping comment")
	}

	return nil
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

	owner := strings.Split(project.Blueprint.Global.Repo.Name, "/")[0]
	repo := strings.Split(project.Blueprint.Global.Repo.Name, "/")[1]
	ghClient, err := github.NewDefaultGithubClient(
		owner,
		repo,
		github.WithCredsOrEnv(project.Blueprint.Global.Ci.Providers.Github.Credentials),
		github.WithLogger(ctx.Logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create github client: %w", err)
	}

	handler := events.NewDefaultEventHandler(ctx.Logger)
	runner := earthly.NewDefaultProjectRunner(ctx, &project)
	return &DocsReleaser{
		config:      config,
		force:       force,
		fs:          fs,
		ghClient:    ghClient,
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

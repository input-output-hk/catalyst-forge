package github

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"path/filepath"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/lib/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

var (
	ErrNoGitHubToken   = fmt.Errorf("no GitHub token found")
	ErrNoRepository    = fmt.Errorf("no repository information found")
	ErrInvalidPR       = fmt.Errorf("invalid pull request number")
	ErrReleaseNotFound = fmt.Errorf("release not found")
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks -out mocks/github.go . GithubClient

// GithubClient is the interface for the Github client.
type GithubClient interface {
	CreateRelease(opts *github.RepositoryRelease) (*github.RepositoryRelease, error)
	Env() *GithubEnv
	GetReleaseByTag(tag string) (*github.RepositoryRelease, error)
	ListPullRequestComments(prNumber int) ([]PullRequestComment, error)
	PostPullRequestComment(prNumber int, body string) error
	ListBranches() ([]Branch, error)
	UploadReleaseAsset(releaseID int64, path string) error
}

// DefaultGithubClient is the default implementation of the Github client.
type DefaultGithubClient struct {
	client      *github.Client
	env         *GithubEnv
	fs          fs.Filesystem
	logger      *slog.Logger
	opts        *DefaultGithubClientOptions
	Owner       string
	RepoName    string
	secretStore *secrets.SecretStore
}

// Branch represents a Git branch in the repository.
type Branch struct {
	Name      string
	CommitSHA string
	Protected bool
}

// PullRequestComment is a comment on a pull request.
type PullRequestComment struct {
	Author string
	Body   string
}

// Env returns the Github environment.
func (g *DefaultGithubClient) Env() *GithubEnv {
	return g.env
}

// CreateRelease creates a new release.
func (g *DefaultGithubClient) CreateRelease(opts *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	g.logger.Info("Creating release", "name", opts.Name)
	release, _, err := g.client.Repositories.CreateRelease(context.Background(), g.Owner, g.RepoName, opts)
	return release, err
}

// ListPullRequestComments lists comments for a pull request.
func (g *DefaultGithubClient) ListPullRequestComments(prNumber int) ([]PullRequestComment, error) {
	var all []PullRequestComment
	ctx := context.Background()
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		g.logger.Debug("Fetching PR comments page",
			"owner", g.Owner, "repo", g.RepoName, "pr", prNumber, "page", opts.Page)
		comments, resp, err := g.client.Issues.ListComments(ctx, g.Owner, g.RepoName, prNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list comments: %w", err)
		}

		for _, c := range comments {
			author := ""
			if c.User != nil && c.User.Login != nil {
				author = *c.User.Login
			}
			body := ""
			if c.Body != nil {
				body = *c.Body
			}
			all = append(all, PullRequestComment{
				Author: author,
				Body:   body,
			})
		}

		// Break out when we've reached the last page.
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return all, nil
}

// GetReleaseByTag gets a release by tag.
func (g *DefaultGithubClient) GetReleaseByTag(tag string) (*github.RepositoryRelease, error) {
	release, resp, err := g.client.Repositories.GetReleaseByTag(context.Background(), g.Owner, g.RepoName, tag)
	if resp.StatusCode == 404 {
		return nil, ErrReleaseNotFound
	}
	return release, err
}

// PostPullRequestComment posts a comment to a pull request.
func (g *DefaultGithubClient) PostPullRequestComment(prNumber int, body string) error {
	if prNumber <= 0 {
		return ErrInvalidPR
	}

	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	comment := &github.IssueComment{
		Body: &body,
	}

	g.logger.Debug("Posting comment to PR", "owner", g.Owner, "repo", g.RepoName, "pr", prNumber)
	_, _, err := g.client.Issues.CreateComment(context.Background(), g.Owner, g.RepoName, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	g.logger.Info("Successfully posted comment to PR", "owner", g.Owner, "repo", g.RepoName, "pr", prNumber)
	return nil
}

// ListBranches lists all branches in the repository.
func (g *DefaultGithubClient) ListBranches() ([]Branch, error) {
	var all []Branch
	ctx := context.Background()
	opts := &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		g.logger.Debug("Fetching branches page",
			"owner", g.Owner, "repo", g.RepoName, "page", opts.Page)
		branches, resp, err := g.client.Repositories.ListBranches(ctx, g.Owner, g.RepoName, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list branches: %w", err)
		}

		for _, b := range branches {
			name := ""
			if b.Name != nil {
				name = *b.Name
			}

			commitSHA := ""
			if b.Commit != nil && b.Commit.SHA != nil {
				commitSHA = *b.Commit.SHA
			}

			all = append(all, Branch{
				Name:      name,
				CommitSHA: commitSHA,
				Protected: b.GetProtected(),
			})
		}

		// Break out when we've reached the last page.
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return all, nil
}

// UploadReleaseAsset uploads a release asset to the given release.
func (g *DefaultGithubClient) UploadReleaseAsset(releaseID int64, path string) error {
	f, err := g.fs.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open release asset: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat asset: %w", err)
	}

	asset := filepath.Base(path)
	contentType := mime.TypeByExtension(filepath.Ext(asset))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	url := fmt.Sprintf("repos/%s/%s/releases/%d/assets?name=%s", g.Owner, g.RepoName, releaseID, asset)
	req, err := g.client.NewUploadRequest(url, f, stat.Size(), contentType)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	_, err = g.client.Do(context.Background(), req, nil)
	if err != nil {
		return fmt.Errorf("failed to upload asset: %w", err)
	}

	return nil
}

// NewDefaultGithubClient creates a new DefaultGithubClient with a *github.Client and repository owner and name.
func NewDefaultGithubClient(owner, repoName string, opts ...DefaultGithubClientOption) (*DefaultGithubClient, error) {
	gc := &DefaultGithubClient{
		opts: &DefaultGithubClientOptions{},
	}
	for _, opt := range opts {
		opt(gc)
	}

	if gc.logger == nil {
		gc.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	if gc.fs == nil {
		gc.fs = billy.NewBaseOsFS()
	}

	if gc.client == nil {
		if gc.opts.Token != "" {
			gc.client = github.NewClient(nil).WithAuthToken(gc.opts.Token)
		} else if gc.opts.Creds != nil {
			if gc.secretStore == nil {
				ss := secrets.NewDefaultSecretStore()
				gc.secretStore = &ss
			}

			creds, err := getGithubProviderCreds(gc.opts.Creds, gc.secretStore, gc.logger)
			if err != nil {
				return nil, fmt.Errorf("could not get Github provider credentials: %w", err)
			}
			gc.client = github.NewClient(nil).WithAuthToken(creds.Token)
		} else {
			gc.client = github.NewClient(nil)
		}
	}

	if gc.env == nil {
		gc.env = &GithubEnv{
			fs:     gc.fs,
			logger: gc.logger,
		}
	}

	return &DefaultGithubClient{
		client:      gc.client,
		env:         gc.env,
		fs:          gc.fs,
		logger:      gc.logger,
		opts:        gc.opts,
		Owner:       owner,
		RepoName:    repoName,
		secretStore: gc.secretStore,
	}, nil
}

package github

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v66/github"
)

var (
	ErrNoGitHubToken = fmt.Errorf("no GitHub token found")
	ErrNoRepository  = fmt.Errorf("no repository information found")
	ErrInvalidPR     = fmt.Errorf("invalid pull request number")
)

// GithubClient is a wrapper around the github client.
type GithubClient struct {
	client   *github.Client
	logger   *slog.Logger
	owner    string
	repoName string
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

// ListPullRequestComments lists comments for a pull request.
func (g *GithubClient) ListPullRequestComments(prNumber int) ([]PullRequestComment, error) {
	var all []PullRequestComment
	ctx := context.Background()
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		g.logger.Debug("Fetching PR comments page",
			"owner", g.owner, "repo", g.repoName, "pr", prNumber, "page", opts.Page)
		comments, resp, err := g.client.Issues.ListComments(ctx, g.owner, g.repoName, prNumber, opts)
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

// PostPullRequestComment posts a comment to a pull request.
func (g *GithubClient) PostPullRequestComment(prNumber int, body string) error {
	if prNumber <= 0 {
		return ErrInvalidPR
	}

	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	comment := &github.IssueComment{
		Body: &body,
	}

	g.logger.Debug("Posting comment to PR", "owner", g.owner, "repo", g.repoName, "pr", prNumber)
	_, _, err := g.client.Issues.CreateComment(context.Background(), g.owner, g.repoName, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	g.logger.Info("Successfully posted comment to PR", "owner", g.owner, "repo", g.repoName, "pr", prNumber)
	return nil
}

// ListBranches lists all branches in the repository.
func (g *GithubClient) ListBranches() ([]Branch, error) {
	var all []Branch
	ctx := context.Background()
	opts := &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		g.logger.Debug("Fetching branches page",
			"owner", g.owner, "repo", g.repoName, "page", opts.Page)
		branches, resp, err := g.client.Repositories.ListBranches(ctx, g.owner, g.repoName, opts)
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

// NewGithubClient creates a new GithubClient with a *github.Client and repository owner and name.
func NewGithubClient(owner, repoName string, client *github.Client, logger *slog.Logger) GithubClient {
	return GithubClient{
		client:   client,
		logger:   logger,
		owner:    owner,
		repoName: repoName,
	}
}

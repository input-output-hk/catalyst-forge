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

// PRClient provides functionality for interacting with pull requests.
type PRClient struct {
	client *github.Client
	logger *slog.Logger
}

// Comment represents the pared‑down information we care about.
type Comment struct {
	Author string
	Body   string
}

// ListComments lists comments for a pull request.
func (p *PRClient) ListComments(owner, repo string, prNumber int) ([]Comment, error) {
	var all []Comment
	ctx := context.Background()
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		p.logger.Debug("Fetching PR comments page",
			"owner", owner, "repo", repo, "pr", prNumber, "page", opts.Page)
		comments, resp, err := p.client.Issues.ListComments(ctx, owner, repo, prNumber, opts)
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
			all = append(all, Comment{
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

// PostComment posts a comment to a pull request.
func (p *PRClient) PostComment(owner, repo string, prNumber int, body string) error {
	if prNumber <= 0 {
		return ErrInvalidPR
	}

	if body == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	comment := &github.IssueComment{
		Body: &body,
	}

	p.logger.Debug("Posting comment to PR", "owner", owner, "repo", repo, "pr", prNumber)
	_, _, err := p.client.Issues.CreateComment(context.Background(), owner, repo, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	p.logger.Info("Successfully posted comment to PR", "owner", owner, "repo", repo, "pr", prNumber)
	return nil
}

// NewPRClient creates a new PRClient with a Github client.
func NewPRClient(client *github.Client, logger *slog.Logger) PRClient {
	return PRClient{
		client: client,
		logger: logger,
	}
}

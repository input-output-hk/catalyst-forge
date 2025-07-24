package github

import (
	"context"
	"fmt"
	"log/slog"
	"os"

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

// CommentOptions contains options for posting a comment.
type CommentOptions struct {
	// Body is the comment content
	Body string
	// CommitID is the specific commit to comment on (optional)
	CommitID string
	// Path is the file path to comment on (optional)
	Path string
	// Position is the line position in the file (optional)
	Position int
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

// NewPRClient creates a new PRClient with authentication from environment.
func NewPRClient(logger *slog.Logger) (*PRClient, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, ErrNoGitHubToken
	}

	client := github.NewClient(nil).WithAuthToken(token)

	return &PRClient{
		client: client,
		logger: logger,
	}, nil
}

// NewCustomPRClient creates a new PRClient with a custom GitHub client.
func NewCustomPRClient(client *github.Client, logger *slog.Logger) *PRClient {
	return &PRClient{
		client: client,
		logger: logger,
	}
}

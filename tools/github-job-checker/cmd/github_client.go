package cmd

import (
	"context"

	"github.com/google/go-github/v66/github"
)

// GitHubClient defines the methods needed from the GitHub API.
type GitHubClient interface {
	FetchCheckRunsForRef(ctx context.Context, owner, repo, ref string) (*github.ListCheckRunsResults, error)
}

// GitHubAPIClient implements GitHubClient using the actual GitHub API.
type GitHubAPIClient struct {
	client *github.Client
}

func NewGitHubAPIClient(token string) *GitHubAPIClient {
	return &GitHubAPIClient{github.NewClient(nil).WithAuthToken(token)}
}

// FetchCheckRunsForRef fetches check runs for a specific reference.
func (c *GitHubAPIClient) FetchCheckRunsForRef(ctx context.Context, owner, repo, ref string) (*github.ListCheckRunsResults, error) {
	results, _, err := c.client.Checks.ListCheckRunsForRef(ctx, owner, repo, ref, nil)
	return results, err
}

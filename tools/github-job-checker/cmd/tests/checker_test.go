package cmd_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/tools/github-job-checker/cmd"
)

// MockGitHubClient mocks the GitHubClient interface for testing.
type MockGitHubClient struct {
	FetchFunc func(ctx context.Context, owner, repo, ref string) (*github.ListCheckRunsResults, error)
}

func (m *MockGitHubClient) FetchCheckRunsForRef(ctx context.Context, owner, repo, ref string) (*github.ListCheckRunsResults, error) {
	if m.FetchFunc != nil {
		return m.FetchFunc(ctx, owner, repo, ref)
	}
	return nil, errors.New("FetchFunc not implemented")
}

func TestChecker_Run_Success(t *testing.T) {
	client := &MockGitHubClient{
		FetchFunc: func(ctx context.Context, owner, repo, ref string) (*github.ListCheckRunsResults, error) {
			return &github.ListCheckRunsResults{
				Total: github.Int(1),
				CheckRuns: []*github.CheckRun{
					{
						Status:     github.String("completed"),
						Conclusion: github.String("success"),
					},
				},
			}, nil
		},
	}

	checker := &cmd.Checker{
		Owner:         "owner",
		Repo:          "repo",
		Ref:           "ref",
		CheckInterval: 1 * time.Second,
		Timeout:       5 * time.Second,
		Client:        client,
	}

	ctx := context.Background()
	err := checker.Run(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestChecker_Run_Failure(t *testing.T) {
	client := &MockGitHubClient{
		FetchFunc: func(ctx context.Context, owner, repo, ref string) (*github.ListCheckRunsResults, error) {
			return &github.ListCheckRunsResults{
				Total: github.Int(1),
				CheckRuns: []*github.CheckRun{
					{
						Status:     github.String("completed"),
						Conclusion: github.String("failure"),
					},
				},
			}, nil
		},
	}

	checker := &cmd.Checker{
		Owner:         "owner",
		Repo:          "repo",
		Ref:           "ref",
		CheckInterval: 1 * time.Second,
		Timeout:       5 * time.Second,
		Client:        client,
	}

	ctx := context.Background()
	err := checker.Run(ctx)
	if err == nil || err.Error() != "one or more GitHub jobs failed" {
		t.Fatalf("expected failure error, got %v", err)
	}
}

func TestChecker_Run_PendingToSuccess(t *testing.T) {
	callCount := 0
	client := &MockGitHubClient{
		FetchFunc: func(ctx context.Context, owner, repo, ref string) (*github.ListCheckRunsResults, error) {
			defer func() { callCount++ }()
			if callCount == 0 {
				// First call: return pending
				return &github.ListCheckRunsResults{
					Total: github.Int(1),
					CheckRuns: []*github.CheckRun{
						{
							Status:     github.String("queued"),
							Conclusion: nil,
						},
					},
				}, nil
			}
			// Subsequent calls: return success
			return &github.ListCheckRunsResults{
				Total: github.Int(1),
				CheckRuns: []*github.CheckRun{
					{
						Status:     github.String("completed"),
						Conclusion: github.String("success"),
					},
				},
			}, nil
		},
	}

	checker := &cmd.Checker{
		Owner:         "owner",
		Repo:          "repo",
		Ref:           "ref",
		CheckInterval: 500 * time.Millisecond,
		Timeout:       2 * time.Second,
		Client:        client,
	}

	ctx := context.Background()
	err := checker.Run(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestChecker_Run_Timeout(t *testing.T) {
	client := &MockGitHubClient{
		FetchFunc: func(ctx context.Context, owner, repo, ref string) (*github.ListCheckRunsResults, error) {
			// Always return in-progress status
			return &github.ListCheckRunsResults{
				Total: github.Int(1),
				CheckRuns: []*github.CheckRun{
					{
						Status:     github.String("in_progress"),
						Conclusion: nil,
					},
				},
			}, nil
		},
	}

	checker := &cmd.Checker{
		Owner:         "owner",
		Repo:          "repo",
		Ref:           "ref",
		CheckInterval: 500 * time.Millisecond,
		Timeout:       1 * time.Second,
		Client:        client,
	}

	ctx := context.Background()
	err := checker.Run(ctx)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) && err.Error() != "timeout of 1s reached. GitHub jobs did not finish in time" {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

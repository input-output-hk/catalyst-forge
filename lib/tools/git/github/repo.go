package github

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks -out mocks/repo.go . GithubRepo

var (
	ErrNotRunningInGHActions = fmt.Errorf("not running in GitHub Actions")
)

// GithubRepo is an interface for interacting with GitHub repositories.
type GithubRepo interface {
	GetBranch() string
	GetCommit() (string, error)
	GetTag() (string, bool)
}

// DefaultGithubRepo is the default implementation of the GithubRepo interface.
type DefaultGithubRepo struct {
	fs     fs.Filesystem
	logger *slog.Logger
}

// GetBranch returns the branch name from the CI environment.
func (g *DefaultGithubRepo) GetBranch() string {
	ref, ok := os.LookupEnv("GITHUB_HEAD_REF")
	if !ok || ref == "" {
		if strings.HasPrefix(os.Getenv("GITHUB_REF"), "refs/heads/") {
			return strings.TrimPrefix(os.Getenv("GITHUB_REF"), "refs/heads/")
		}
	}

	return ref
}

// GetCommit returns the commit SHA from the CI environment.
func (g *DefaultGithubRepo) GetCommit() (string, error) {
	if !InGithubActions() {
		return "", ErrNotRunningInGHActions
	}

	if g.getEventType() == "pull_request" {
		g.logger.Debug("Found GitHub pull request event")
		event, err := g.getEventPayload()
		if err != nil {
			return "", fmt.Errorf("failed to get event payload: %w", err)
		}

		pr, ok := event.(*github.PullRequestEvent)
		if !ok {
			return "", fmt.Errorf("unexpected event type")
		}

		if pr.PullRequest.Head.SHA == nil {
			return "", fmt.Errorf("pull request head SHA is empty")
		}

		return *pr.PullRequest.Head.SHA, nil
	} else if g.getEventType() == "push" {
		g.logger.Debug("Found GitHub push event")
		event, err := g.getEventPayload()
		if err != nil {
			return "", fmt.Errorf("failed to get event payload: %w", err)
		}

		push, ok := event.(*github.PushEvent)
		if !ok {
			return "", fmt.Errorf("unexpected event type")
		}

		if push.After == nil {
			return "", fmt.Errorf("push event after SHA is empty")
		}

		return *push.After, nil
	}

	return "", fmt.Errorf("unsupported event type: %s", g.getEventType())
}

// GetTag returns the tag from the CI environment if it exists.
// If the tag is not found, it returns an empty string and false.
func (g *DefaultGithubRepo) GetTag() (string, bool) {
	tag, exists := os.LookupEnv("GITHUB_REF")
	if exists && strings.HasPrefix(tag, "refs/tags/") {
		return strings.TrimPrefix(tag, "refs/tags/"), true
	}

	return "", false
}

// getEventPayload returns the GitHub event payload.
func (g *DefaultGithubRepo) getEventPayload() (any, error) {
	if !InGithubActions() {
		return "", ErrNotRunningInGHActions
	}

	path := os.Getenv("GITHUB_EVENT_PATH")
	name := os.Getenv("GITHUB_EVENT_NAME")

	g.logger.Debug("Reading GitHub event data", "path", path, "name", name)
	payload, err := g.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read GitHub event data: %w", err)
	}

	event, err := github.ParseWebHook(name, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub event data: %w", err)
	}

	return event, nil
}

// getEventType returns the GitHub event type.
func (g *DefaultGithubRepo) getEventType() string {
	return os.Getenv("GITHUB_EVENT_NAME")
}

// NewDefaultGithubRepo creates a new DefaultGithubRepo.
func NewDefaultGithubRepo(logger *slog.Logger) DefaultGithubRepo {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return DefaultGithubRepo{
		fs:     billy.NewBaseOsFS(),
		logger: logger,
	}
}

// NewCustomGithubRepo creates a new DefaultGithubRepo with a custom filesystem.
func NewCustomDefaultGithubRepo(fs fs.Filesystem, logger *slog.Logger) DefaultGithubRepo {
	return DefaultGithubRepo{
		fs:     fs,
		logger: logger,
	}
}

// InGithubActions returns whether the current process is running in a GitHub Actions environment.
func InGithubActions() bool {
	_, pathExists := os.LookupEnv("GITHUB_EVENT_PATH")
	_, nameExists := os.LookupEnv("GITHUB_EVENT_NAME")
	return pathExists && nameExists
}

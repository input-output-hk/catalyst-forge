package github

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

var (
	ErrNoEventFound = fmt.Errorf("no GitHub event data found")
	ErrTagNotFound  = fmt.Errorf("tag not found")
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks -out mocks/env.go . GithubEnv

// GithubEnv provides GitHub environment information.
type GithubEnv interface {
	GetBranch() string
	GetEventPayload() (any, error)
	GetEventType() string
	GetPRNumber() int
	GetTag() string
	IsPR() bool
	HasEvent() bool
}

// DefaultGithubEnv provides the default implementation of the GithubEnv interface.
type DefaultGithubEnv struct {
	fs     fs.Filesystem
	logger *slog.Logger
}

// GetBranch returns the current branch from the CI environment.
func (g *DefaultGithubEnv) GetBranch() string {
	ref, ok := os.LookupEnv("GITHUB_HEAD_REF")
	if !ok || ref == "" {
		if strings.HasPrefix(os.Getenv("GITHUB_REF"), "refs/heads/") {
			return strings.TrimPrefix(os.Getenv("GITHUB_REF"), "refs/heads/")
		}
	}

	return ref
}

// GetEventPayload returns the GitHub event payload.
func (g *DefaultGithubEnv) GetEventPayload() (any, error) {
	path, pathExists := os.LookupEnv("GITHUB_EVENT_PATH")
	name, nameExists := os.LookupEnv("GITHUB_EVENT_NAME")

	if !pathExists || !nameExists {
		return nil, ErrNoEventFound
	}

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

// GetEventType returns the GitHub event type.
func (g *DefaultGithubEnv) GetEventType() string {
	return os.Getenv("GITHUB_EVENT_NAME")
}

// GetTag returns the tag from the CI environment if it exists.
// If the tag is not found, an empty string is returned.
func (g *DefaultGithubEnv) GetTag() string {
	tag, exists := os.LookupEnv("GITHUB_REF")
	if exists && strings.HasPrefix(tag, "refs/tags/") {
		return strings.TrimPrefix(tag, "refs/tags/")
	}

	return ""
}

// GetPRNumber returns the pull request number if the current environment is associated with a PR.
// Returns 0 if not in a PR context or if the PR number cannot be determined.
func (g *DefaultGithubEnv) GetPRNumber() int {
	if !g.IsPR() {
		return 0
	}

	if prNumberStr, ok := os.LookupEnv("GITHUB_EVENT_NUMBER"); ok {
		if prNumber, err := strconv.Atoi(prNumberStr); err == nil {
			return prNumber
		}
	}

	if g.HasEvent() {
		event, err := g.GetEventPayload()
		if err != nil {
			g.logger.Debug("Failed to get event payload for PR number", "error", err)
			return 0
		}

		if prEvent, ok := event.(*github.PullRequestEvent); ok {
			if prEvent.PullRequest != nil && prEvent.PullRequest.Number != nil {
				return *prEvent.PullRequest.Number
			}
		}
	}

	return 0
}

// IsPR returns whether the current environment is associated with a pull request.
func (g *DefaultGithubEnv) IsPR() bool {
	return g.GetEventType() == "pull_request"
}

// HasEvent returns whether a GitHub event payload exists.
func (g *DefaultGithubEnv) HasEvent() bool {
	_, pathExists := os.LookupEnv("GITHUB_EVENT_PATH")
	_, nameExists := os.LookupEnv("GITHUB_EVENT_NAME")
	return pathExists && nameExists
}

// InCI returns whether the code is running in a CI environment.
func InCI() bool {
	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		return true
	}

	return false
}

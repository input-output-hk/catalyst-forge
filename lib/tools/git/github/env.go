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

var (
	ErrNoEventFound = fmt.Errorf("no GitHub event data found")
	ErrTagNotFound  = fmt.Errorf("tag not found")
)

// GithubEnv provides GitHub environment information.
type GithubEnv struct {
	fs     fs.Filesystem
	logger *slog.Logger
}

func (g *GithubEnv) GetBranch() string {
	ref, ok := os.LookupEnv("GITHUB_HEAD_REF")
	if !ok || ref == "" {
		if strings.HasPrefix(os.Getenv("GITHUB_REF"), "refs/heads/") {
			return strings.TrimPrefix(os.Getenv("GITHUB_REF"), "refs/heads/")
		}
	}

	return ref
}

// GetEventPayload returns the GitHub event payload.
func (g *GithubEnv) GetEventPayload() (any, error) {
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
func (g *GithubEnv) GetEventType() string {
	return os.Getenv("GITHUB_EVENT_NAME")
}

// GetTag returns the tag from the CI environment if it exists.
// If the tag is not found, an empty string is returned.
func (g *GithubEnv) GetTag() string {
	tag, exists := os.LookupEnv("GITHUB_REF")
	if exists && strings.HasPrefix(tag, "refs/tags/") {
		return strings.TrimPrefix(tag, "refs/tags/")
	}

	return ""
}

// HasEvent returns whether a GitHub event payload exists.
func (g *GithubEnv) HasEvent() bool {
	_, pathExists := os.LookupEnv("GITHUB_EVENT_PATH")
	_, nameExists := os.LookupEnv("GITHUB_EVENT_NAME")
	return pathExists && nameExists
}

// NewGithubEnv creates a new GithubEnv.
func NewGithubEnv(logger *slog.Logger) GithubEnv {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return GithubEnv{
		fs:     billy.NewBaseOsFS(),
		logger: logger,
	}
}

// NewCustomGithubEnv creates a new GithubEnv with a custom filesystem.
func NewCustomGithubEnv(fs fs.Filesystem, logger *slog.Logger) GithubEnv {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return GithubEnv{
		fs:     fs,
		logger: logger,
	}
}

// InCI returns whether the code is running in a CI environment.
func InCI() bool {
	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		return true
	}

	return false
}

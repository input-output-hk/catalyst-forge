package project

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/tools/earthly"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
)

// Project represents a project
type Project struct {
	// Blueprint is the project blueprint.
	Blueprint sb.Blueprint

	// Earthfile is the project Earthfile.
	Earthfile *earthly.Earthfile

	// Name is the project name.
	Name string

	// Path is the project path.
	Path string

	// RawBlueprint is the raw blueprint.
	RawBlueprint blueprint.RawBlueprint

	// Repo is the project git repository.
	Repo *repo.GitRepo

	// RepoRoot is the path to the repository root.
	RepoRoot string

	// SecretStore is the project secret store.
	SecretStore secrets.SecretStore

	// Tag is the project tag, if it exists in the current context.
	Tag *ProjectTag

	// TagInfo is the project tag information.
	//TagInfo *TagInfo

	ctx    *cue.Context
	logger *slog.Logger
}

// GetRelativePath returns the relative path of the project from the repo root.
func (p *Project) GetRelativePath() (string, error) {
	var projectPath, repoRoot string
	var err error

	if !filepath.IsAbs(p.Path) {
		projectPath, err = filepath.Abs(p.Path)
		if err != nil {
			return "", fmt.Errorf("failed to get project path: %w", err)
		}
	} else {
		projectPath = p.Path
	}

	if !filepath.IsAbs(p.RepoRoot) {
		repoRoot, err = filepath.Abs(p.RepoRoot)
		if err != nil {
			return "", fmt.Errorf("failed to get repo root: %w", err)
		}
	} else {
		repoRoot = p.RepoRoot
	}

	if !strings.HasPrefix(projectPath, repoRoot) {
		return "", fmt.Errorf("project path is not a subdirectory of the repo root")
	}

	relPath, err := filepath.Rel(repoRoot, projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	return relPath, nil
}

// GetDeploymentEvents returns the deployment events for a project.
func (p *Project) GetDeploymentEvents() map[string]cue.Value {
	events := make(map[string]cue.Value)
	for event := range p.Blueprint.Project.Deployment.On {
		config := p.RawBlueprint.Get(fmt.Sprintf("project.deployment.on.%s", event))
		events[event] = config
	}

	return events
}

// GetReleaseEvents returns the release events for a release.
func (p *Project) GetReleaseEvents(releaseName string) map[string]cue.Value {
	release, ok := p.Blueprint.Project.Release[releaseName]
	if !ok {
		return nil
	}

	events := make(map[string]cue.Value)
	for event := range release.On {
		config := p.RawBlueprint.Get(fmt.Sprintf("project.release.%s.on.%s", releaseName, event))
		events[event] = config
	}

	return events
}

// Raw returns the raw blueprint.
func (p *Project) Raw() blueprint.RawBlueprint {
	return p.RawBlueprint
}

// NewProject creates a new project.
func NewProject(
	ctx *cue.Context,
	repo *repo.GitRepo,
	earthfile *earthly.Earthfile,
	name, path, repoRoot string,
	blueprint sb.Blueprint,
	tag *ProjectTag,
	logger *slog.Logger,
	secretStore secrets.SecretStore,
) Project {
	return Project{
		Blueprint:   blueprint,
		Earthfile:   earthfile,
		Name:        name,
		Path:        path,
		Repo:        repo,
		RepoRoot:    repoRoot,
		SecretStore: secretStore,
		Tag:         tag,
		ctx:         ctx,
		logger:      logger,
	}
}

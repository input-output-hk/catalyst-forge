package project

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	gg "github.com/go-git/go-git/v5"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/earthfile"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git"
)

// TagInfo represents tag information.
type TagInfo struct {
	// Generated is the generated tag.
	Generated git.Tag `json:"generated"`

	// Git is the git tag.
	Git git.Tag `json:"git"`
}

// Project represents a project
type Project struct {
	// Blueprint is the project blueprint.
	Blueprint schema.Blueprint

	// Earthfile is the project Earthfile.
	Earthfile *earthfile.Earthfile

	// Name is the project name.
	Name string

	// Path is the project path.
	Path string

	// RawBlueprint is the raw blueprint.
	RawBlueprint blueprint.RawBlueprint

	// Repo is the project git repository.
	Repo *gg.Repository

	// RepoRoot is the path to the repository root.
	RepoRoot string

	// TagInfo is the project tag information.
	TagInfo *TagInfo

	logger *slog.Logger
	ctx    *cue.Context
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

// TagMatches checks if the git tag matches the project.
func (p *Project) TagMatches() (bool, error) {
	if p.TagInfo.Git == "" {
		p.logger.Debug("No git tag found")
		return false, nil
	} else if !p.TagInfo.Git.IsMono() {
		p.logger.Debug("Found regular tag", "tag", p.TagInfo.Git)
		return true, nil
	}

	relPath, err := p.GetRelativePath()
	if err != nil {
		return false, err
	}

	mtag := p.TagInfo.Git.ToMono()
	p.logger.Debug("Found mono tag", "tag", mtag.Full, "project", mtag.Project, "tag", mtag.Tag)

	if relPath == mtag.Project {
		p.logger.Debug("Tag matches project")
		return true, nil
	}

	alias, ok := p.Blueprint.Global.CI.Tagging.Aliases[mtag.Project]
	if ok && relPath == alias {
		p.logger.Debug("Tag matches alias", "alias", alias)
		return true, nil
	}

	p.logger.Debug("Tag does not match project")
	return false, nil
}

// Raw returns the raw blueprint.
func (p *Project) Raw() blueprint.RawBlueprint {
	return p.RawBlueprint
}

func NewProject(
	logger *slog.Logger,
	ctx *cue.Context,
	repo *gg.Repository,
	earthfile *earthfile.Earthfile,
	name, path, repoRoot string,
	blueprint schema.Blueprint,
	tagInfo *TagInfo,
) Project {
	return Project{
		Blueprint: blueprint,
		Earthfile: earthfile,
		Name:      name,
		Path:      path,
		Repo:      repo,
		RepoRoot:  repoRoot,
		TagInfo:   tagInfo,
		ctx:       ctx,
		logger:    logger,
	}
}

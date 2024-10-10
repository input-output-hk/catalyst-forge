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

	// Repo is the project git repository.
	Repo *gg.Repository

	// RepoRoot is the path to the repository root.
	RepoRoot string

	// TagInfo is the project tag information.
	TagInfo TagInfo

	logger       *slog.Logger
	ctx          *cue.Context
	rawBlueprint blueprint.RawBlueprint
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

// MatchesTag returns true if the project matches the given tag.
func (p *Project) MatchesTag(tag git.MonoTag) (bool, error) {
	relPath, err := p.GetRelativePath()
	if err != nil {
		return false, err
	}

	if relPath == tag.Project {
		return true, nil
	}

	alias, ok := p.Blueprint.Global.CI.Tagging.Aliases[tag.Project]
	if ok && relPath == alias {
		return true, nil
	}

	return false, nil
}

// Raw returns the raw blueprint.
func (p *Project) Raw() blueprint.RawBlueprint {
	return p.rawBlueprint
}

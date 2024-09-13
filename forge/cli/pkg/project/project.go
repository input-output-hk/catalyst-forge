package project

import (
	"fmt"
	"path/filepath"
	"strings"

	gg "github.com/go-git/go-git/v5"
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/blueprint"
	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/earthfile"
)

type TagInfo struct {
	Generated string `json:"generated"`
	Git       string `json:"git"`
}

// Project represents a project
type Project struct {
	Blueprint    schema.Blueprint
	Earthfile    *earthfile.Earthfile
	Name         string
	Path         string
	Repo         *gg.Repository
	RepoRoot     string
	Tags         TagInfo
	rawBlueprint blueprint.RawBlueprint
}

// Raw returns the raw blueprint.
func (p *Project) Raw() blueprint.RawBlueprint {
	return p.rawBlueprint
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

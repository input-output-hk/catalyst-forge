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
)

// Project represents a project
type Project struct {
	Blueprint    schema.Blueprint
	ctx          *cue.Context
	Earthfile    *earthfile.Earthfile
	Name         string
	Path         string
	Repo         *gg.Repository
	RepoRoot     string
	TagInfo      TagInfo
	logger       *slog.Logger
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

// Raw returns the raw blueprint.
func (p *Project) Raw() blueprint.RawBlueprint {
	return p.rawBlueprint
}

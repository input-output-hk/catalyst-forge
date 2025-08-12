package scan

import (
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/scan"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

// scanProjects scans for projects in the given root path.
func scanProjects(ctx run.RunContext, rootPath string, absolute bool) (map[string]project.Project, error) {
	if rootPath == "" {
		rootPath = "."
	}

	projects, err := scan.ScanProjects(rootPath, ctx.ProjectLoader, &ctx.FSWalker, ctx.Logger)
	if err != nil {
		return nil, err
	}

	if !absolute {
		normalizedProjects := make(map[string]project.Project)
		for path, project := range projects {
			relPath, err := filepath.Rel(rootPath, path)
			if err != nil {
				return nil, err
			}
			normalizedProjects[relPath] = project
		}
		return normalizedProjects, nil
	}

	return projects, nil
}

package scan

import (
	"fmt"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/scan"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

// getAbsolutePath returns the absolute path for the given path.
func getAbsolutePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	return filepath.Abs(path)
}

// scanProjects scans the projects in the given root path.
func scanProjects(ctx run.RunContext, rootPath string, absolute bool) (map[string]project.Project, error) {
	var err error

	if absolute {
		rootPath, err = getAbsolutePath(rootPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	exists, err := fs.Exists(rootPath)
	if err != nil {
		return nil, fmt.Errorf("could not check if root path exists: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("root path does not exist: %s", rootPath)
	}

	return scan.ScanProjects(rootPath, ctx.ProjectLoader, &ctx.FSWalker, ctx.Logger)
}

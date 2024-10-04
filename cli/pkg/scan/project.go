package scan

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/input-output-hk/catalyst-forge/lib/project/pkg/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/walker"
)

// ScanProjects scans the given root path for projects and returns a map of
// project paths to the projects themselves.
func ScanProjects(rootPath string, l project.ProjectLoader, w walker.Walker, logger *slog.Logger) (map[string]project.Project, error) {
	projects := make(map[string]project.Project)
	err := w.Walk(rootPath, func(path string, fileType walker.FileType, openFile func() (walker.FileSeeker, error)) error {
		if fileType != walker.FileTypeFile {
			return nil
		} else if filepath.Base(path) != "blueprint.cue" {
			return nil
		}

		path = filepath.Dir(path)

		// We need to drop the blueprint suffix and make sure relative paths
		// include a leading "./" to avoid confusing the Earthly CLI
		if !strings.HasPrefix(rootPath, "/") && path != "." {
			path = fmt.Sprintf("./%s", path)
		}

		logger.Info("loading project", "path", path, "rootPath", rootPath)
		p, err := l.Load(path)
		if err != nil {
			logger.Error("error loading project", "path", path, "error", err)
			return fmt.Errorf("error loading %s: %w", path, err)
		}

		projects[path] = p

		return nil
	})

	return projects, err
}

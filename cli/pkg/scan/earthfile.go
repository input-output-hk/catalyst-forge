package scan

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/earthfile"
	w "github.com/input-output-hk/catalyst-forge/lib/tools/pkg/walker"
)

// ScanEarthfiles scans the given root path for Earthfiles and returns a map
// that maps the path of the Earthfile to the targets defined in the Earthfile.
func ScanEarthfiles(rootPath string, walker w.Walker, logger *slog.Logger) (map[string]earthfile.Earthfile, error) {
	earthfiles := make(map[string]earthfile.Earthfile)

	err := walker.Walk(rootPath, func(path string, fileType w.FileType, openFile func() (w.FileSeeker, error)) error {
		if fileType != w.FileTypeFile {
			return nil
		} else if filepath.Base(path) != "Earthfile" {
			return nil
		}

		file, err := openFile()
		if err != nil {
			return err
		}
		defer file.Close()

		logger.Info("parsing Earthfile", "path", path)
		earthfile, err := earthfile.ParseEarthfile(context.Background(), file)
		if err != nil {
			logger.Error("error parsing Earthfile", "path", path, "error", err)
			return fmt.Errorf("error parsing %s: %w", path, err)
		}

		// We need to drop the Earthfile suffix and make sure relative paths
		// include a leading "./" to avoid confusing the Earthly CLI
		path = filepath.Dir(path)
		if !strings.HasPrefix(rootPath, "/") && path != "." {
			path = fmt.Sprintf("./%s", path)
		}

		earthfiles[path] = earthfile

		return nil
	})

	return earthfiles, err
}

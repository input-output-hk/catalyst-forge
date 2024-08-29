package earthfile

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	w "github.com/input-output-hk/catalyst-forge/forge/cli/pkg/walker"
)

// ScanEarthfiles scans the given root path for Earthfiles and returns a map
// that maps the path of the Earthfile to the targets defined in the Earthfile.
func ScanEarthfiles(rootPath string, walker w.Walker, logger *slog.Logger) (map[string]Earthfile, error) {
	earthfiles := make(map[string]Earthfile)

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
		earthfile, err := ParseEarthfile(context.Background(), file)
		if err != nil {
			logger.Error("error parsing Earthfile", "path", path, "error", err)
			return fmt.Errorf("error parsing %s: %w", path, err)
		}

		earthfiles[path] = earthfile

		return nil
	})

	return earthfiles, err
}
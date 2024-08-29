package walker

import (
	"log/slog"
	"os"

	"github.com/spf13/afero"
)

// FilesystemWalker is a walker that walks over the local filesystem.
type FilesystemWalker struct {
	fs     afero.Fs
	logger *slog.Logger
}

// Walk walks over the files and directories in the given root path and calls
// the given function for each entry.
// The reader passed to the function is closed after the function returns.
func (w *FilesystemWalker) Walk(rootPath string, callback WalkerCallback) error {
	return afero.Walk(w.fs, rootPath, func(path string, info os.FileInfo, err error) error {
		w.logger.Debug("walking path", "path", path)
		if err != nil {
			w.logger.Error("error walking path", "path", path, "error", err)
			return err
		}

		if info.Mode().IsRegular() {
			return callback(path, FileTypeFile, func() (FileSeeker, error) {
				w.logger.Debug("opening file", "path", path)
				reader, err := w.fs.Open(path)
				if err != nil {
					w.logger.Error("error opening file", "path", path, "error", err)
					return nil, err
				}

				return reader, nil
			})
		} else if info.Mode().IsDir() {
			return callback(path, FileTypeDir, func() (FileSeeker, error) {
				return nil, nil
			})
		} else {
			return nil
		}
	})
}

// NewFilesystemWalker creates a new FilesystemWalker.
func NewFilesystemWalker(logger *slog.Logger) FilesystemWalker {
	return FilesystemWalker{
		fs:     afero.NewOsFs(),
		logger: logger,
	}
}

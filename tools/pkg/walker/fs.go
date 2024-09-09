package walker

import (
	"log/slog"
	"os"

	"github.com/spf13/afero"
)

// FilesystemWalker is a walker that walks over the local filesystem.
type FSWalker struct {
	fs     afero.Fs
	logger *slog.Logger
}

// Walk walks over the files and directories in the given root path and calls
// the given function for each entry.
// The reader passed to the function is closed after the function returns.
func (w *FSWalker) Walk(rootPath string, callback WalkerCallback) error {
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

// NewFilesystemWalker creates a new FSWalker with the given filesystem.
func NewFSWalker(fs afero.Fs, logger *slog.Logger) FSWalker {
	return FSWalker{
		fs:     fs,
		logger: logger,
	}
}

// NewDefaultFilesystemWalker creates a new FSWalker with the default filesystem
// and an optional logger.
func NewDefaultFSWalker(logger *slog.Logger) FSWalker {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	return FSWalker{
		fs:     afero.NewOsFs(),
		logger: logger,
	}
}

// NewCustomDefaultFilesystemWalker creates a new FSWalker with the given
// filesystem and an optional logger.
func NewCustomDefaultFSWalker(fs afero.Fs, logger *slog.Logger) FSWalker {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	return FSWalker{
		fs:     fs,
		logger: logger,
	}
}

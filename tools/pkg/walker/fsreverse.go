package walker

//go:generate go run github.com/matryer/moq@latest -out ./walker_mock.go . ReverseWalker

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// FSReverseWalker is a ReverseWalker that walks over the local filesystem.
type FSReverseWalker struct {
	fs     afero.Fs
	logger *slog.Logger
}

// Walk performs a reverse walk over the files and directories from the start
// path to the end path and calls the given function for each entry.
func (w *FSReverseWalker) Walk(startPath, endPath string, callback WalkerCallback) error {
	currentDir, err := filepath.Abs(startPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	endDir, err := filepath.Abs(endPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if !strings.HasPrefix(currentDir, endDir) {
		return fmt.Errorf("start path is not a subdirectory of end path")
	}

	for {
		w.logger.Debug("reverse walking path", "path", currentDir)
		files, err := afero.ReadDir(w.fs, currentDir)

		if err != nil {
			w.logger.Error("error reading directory", "path", currentDir, "error", err)
			return fmt.Errorf("failed to read directory: %w", err)
		}

		for _, file := range files {
			path := filepath.Join(currentDir, file.Name())

			if file.IsDir() {
				err := callback(path, FileTypeDir, func() (FileSeeker, error) {
					return nil, nil
				})

				if errors.Is(err, io.EOF) {
					return nil
				} else if err != nil {
					return err
				}
			} else if file.Mode().IsRegular() {
				err := callback(path, FileTypeFile, func() (FileSeeker, error) {
					handle, err := w.fs.Open(path)

					if err != nil {
						w.logger.Error("error opening file", "path", path, "error", err)
						return nil, fmt.Errorf("failed to open file: %w", err)
					}

					return handle, nil
				})

				if errors.Is(err, io.EOF) {
					return nil
				} else if err != nil {
					return err
				}
			}
		}

		if currentDir == endDir {
			return nil
		} else {
			currentDir = filepath.Dir(currentDir)
		}
	}
}

// NewFSReverseWalker creates a new FSReverseWalker with default
// settings and an optional logger.
func NewDefaultFSReverseWalker(logger *slog.Logger) FSReverseWalker {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return FSReverseWalker{
		fs:     afero.NewOsFs(),
		logger: logger,
	}
}

// NewFSReverseWalker creates a new FSReverseWalker with an
// optional logger.
func NewFSReverseWalker(logger *slog.Logger, fs afero.Fs) FSReverseWalker {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return FSReverseWalker{
		fs:     fs,
		logger: logger,
	}
}

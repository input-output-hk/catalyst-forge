package loader

//go:generate go run github.com/matryer/moq@latest -out ./walker_mock.go . ReverseWalker

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// FileType is an enum that represents the type of a file.
type FileType int

const (
	FileTypeFile FileType = iota
	FileTypeDir
)

// FileSeeker is an interface that combines the fs.File and io.Seeker interfaces.
type FileSeeker interface {
	fs.File
	io.Seeker
}

// WalkerCallback is a callback function that is called for each file in the
// Walk function.
type WalkerCallback func(string, FileType, func() (FileSeeker, error)) error

// ReverseWalker is an interface that allows reverse walking over a set of
// files.
// The start path is the path where the walk starts and the end path is the path
// where the walk ends.
type ReverseWalker interface {
	// Walk performs a reverse walk from the end path to the start path and
	// calls the given function for each file.
	Walk(startPath, endPath string, callback WalkerCallback) error
}

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

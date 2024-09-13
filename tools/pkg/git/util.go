package git

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker"
)

var (
	ErrGitRootNotFound = errors.New("git root not found")
)

// FindGitRoot finds the root of a Git repository starting from the given
// path. It returns the path to the root of the Git repository or an error if
// the root is not found.
func FindGitRoot(startPath string, rw walker.ReverseWalker) (string, error) {
	var gitRoot string
	err := rw.Walk(
		startPath,
		"/",
		func(path string, fileType walker.FileType, openFile func() (walker.FileSeeker, error)) error {
			if fileType == walker.FileTypeDir {
				if filepath.Base(path) == ".git" {
					gitRoot = filepath.Dir(path)
					return io.EOF
				}
			}

			return nil
		},
	)

	if err != nil {
		return "", err
	}

	if gitRoot == "" {
		return "", ErrGitRootNotFound
	}

	return gitRoot, nil
}

// InCI returns true if the code is running in a CI environment.
func InCI() bool {
	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		return true
	}

	return false
}

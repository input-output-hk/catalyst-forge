package git

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
)

var (
	ErrGitRootNotFound = errors.New("git root not found")
)

// TagObjectsToMap converts a slice of tag objects into a map where the key is the tag name
// and the value is the corresponding tag object.
func TagObjectsToMap(tags []*object.Tag) map[string]*object.Tag {
	tagMap := make(map[string]*object.Tag, len(tags))
	for _, tag := range tags {
		tagMap[tag.Name] = tag
	}
	return tagMap
}

// GetTagCommit returns the commit that the given tag object points to.
func GetTagCommit(tag *object.Tag) (*object.Commit, error) {
	if tag == nil {
		return nil, fmt.Errorf("tag is nil")
	}
	return tag.Commit()
}

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

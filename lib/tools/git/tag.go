package git

import (
	"fmt"
	"os"
	"strings"

	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
)

var (
	ErrTagNotFound = fmt.Errorf("tag not found")
)

// GetTag returns the tag of the current HEAD commit.
func GetTag(r *repo.GitRepo) (string, error) {
	var tag string
	var err error
	if InCI() {
		tag, err = getCITag()
		if err != nil {
			return "", err
		}
	} else {
		tag, err = r.GetCurrentTag()
		if err != nil {
			return "", err
		}
	}

	return tag, nil
}

// getCITag returns the tag from the CI environment if it exists.
func getCITag() (string, error) {
	tag, exists := os.LookupEnv("GITHUB_REF")
	if exists && strings.HasPrefix(tag, "refs/tags/") {
		return strings.TrimPrefix(tag, "refs/tags/"), nil
	} else {
		return "", ErrTagNotFound
	}
}

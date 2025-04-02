package git

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/lib/tools/git/github"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
)

var (
	ErrTagNotFound = fmt.Errorf("tag not found")
)

// GetTag returns the tag of the current HEAD commit.
func GetTag(r *repo.GitRepo) (string, error) {
	var tag string
	var err error
	ghr := github.NewDefaultGithubRepo(nil)

	if github.InGithubActions() {
		var ok bool
		tag, ok = ghr.GetTag()
		if !ok {
			return "", ErrTagNotFound
		}
	} else {
		tag, err = r.GetCurrentTag()
		if err != nil {
			return "", err
		}
	}

	return tag, nil
}

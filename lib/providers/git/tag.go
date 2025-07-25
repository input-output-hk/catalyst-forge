package git

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/lib/providers/github"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
)

var (
	ErrTagNotFound = fmt.Errorf("tag not found")
)

// GetTag returns the tag of the current HEAD commit.
func GetTag(r *repo.GitRepo) (string, error) {
	var tag string
	var err error
	gc, err := github.NewDefaultGithubClient("", "")
	if err != nil {
		return "", fmt.Errorf("failed to create github client: %w", err)
	}

	if github.InCI() {
		tag = gc.Env().GetTag()
		if tag == "" {
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

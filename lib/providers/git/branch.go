package git

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/lib/providers/github"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
)

var (
	ErrBranchNotFound = fmt.Errorf("branch not found")
)

func GetBranch(repo *repo.GitRepo) (string, error) {
	gc, err := github.NewDefaultGithubClient("", "")
	if err != nil {
		return "", fmt.Errorf("failed to create github client: %w", err)
	}

	if github.InCI() {
		ref := gc.Env().GetBranch()
		if ref != "" {
			return ref, nil
		}
	}

	branch, err := repo.GetCurrentBranch()
	if err != nil {
		return "", err
	}

	return branch, nil
}

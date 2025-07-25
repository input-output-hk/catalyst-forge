package git

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/lib/providers/github"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
)

var (
	ErrBranchNotFound = fmt.Errorf("branch not found")
)

func GetBranch(gc github.GithubClient, repo *repo.GitRepo) (string, error) {
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

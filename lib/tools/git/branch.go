package git

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/lib/tools/git/github"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
)

var (
	ErrBranchNotFound = fmt.Errorf("branch not found")
)

func GetBranch(repo *repo.GitRepo) (string, error) {
	env := github.NewGithubEnv(nil)

	if github.InCI() {
		ref := env.GetBranch()
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

package git

import (
	"fmt"
	"os"
	"strings"

	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
)

var (
	ErrBranchNotFound = fmt.Errorf("branch not found")
)

func GetBranch(repo *repo.GitRepo) (string, error) {
	if InCI() {
		ref, ok := os.LookupEnv("GITHUB_HEAD_REF")
		if !ok || ref == "" {
			if strings.HasPrefix(os.Getenv("GITHUB_REF"), "refs/heads/") {
				return strings.TrimPrefix(os.Getenv("GITHUB_REF"), "refs/heads/"), nil
			}

			// Revert to trying to get the branch from the local repository
		} else {
			return ref, nil
		}
	}

	branch, err := repo.GetCurrentBranch()
	if err != nil {
		return "", err
	}

	return branch, nil
}

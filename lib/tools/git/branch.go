package git

import (
	"fmt"
	"os"
	"strings"

	gg "github.com/go-git/go-git/v5"
)

var (
	ErrBranchNotFound = fmt.Errorf("branch not found")
)

func GetBranch(repo *gg.Repository) (string, error) {
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

	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	return ref.Name().Short(), nil
}

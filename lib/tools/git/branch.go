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
		fmt.Println("Running in CI")
		ref, ok := os.LookupEnv("GITHUB_HEAD_REF")
		if !ok {
			if strings.HasPrefix(os.Getenv("GITHUB_REF"), "refs/heads/") {
				fmt.Printf("Branch from GITHUB_REF: %s\n", os.Getenv("GITHUB_REF"))
				fmt.Println(strings.TrimPrefix(os.Getenv("GITHUB_REF"), "refs/heads/"))
				return strings.TrimPrefix(os.Getenv("GITHUB_REF"), "refs/heads/"), nil
			}

			fmt.Println("Found no CI branch")

			// Revert to trying to get the branch from the local repository
		} else {
			fmt.Printf("Branch from GITHUB_HEAD_REF: %s\n", ref)
			return ref, nil
		}
	}

	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	return ref.Name().Short(), nil
}

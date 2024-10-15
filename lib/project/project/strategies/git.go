package strategies

import (
	"fmt"

	"github.com/go-git/go-git/v5"
)

// GitCommit returns the commit hash of the HEAD commit.
func GitCommit(repo *git.Repository) (string, error) {
	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	obj, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit object: %w", err)
	}

	return obj.Hash.String(), nil
}

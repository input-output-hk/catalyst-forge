package tag

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
)

// GitCommit returns the commit hash of the HEAD commit.
func GitCommit(project *project.Project) (string, error) {
	ref, err := project.Repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	obj, err := project.Repo.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit object: %w", err)
	}

	return obj.Hash.String(), nil
}

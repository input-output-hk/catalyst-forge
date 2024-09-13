package tag

import (
	"testing"

	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/project"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestGitCommit(t *testing.T) {
	repo := testutils.NewInMemRepo(t)
	repo.AddFile(t, "example.txt", "example content")
	commit := repo.Commit(t, "Initial commit")
	project := project.Project{
		Repo: repo.Repo,
	}

	commitHash, err := GitCommit(&project)
	assert.NoError(t, err)
	assert.Equal(t, commit.String(), commitHash)
}

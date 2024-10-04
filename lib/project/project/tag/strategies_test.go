package tag

import (
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
)

func TestGitCommit(t *testing.T) {
	repo := testutils.NewInMemRepo(t)
	repo.AddFile(t, "example.txt", "example content")
	commit := repo.Commit(t, "Initial commit")

	commitHash, err := GitCommit(repo.Repo)
	assert.NoError(t, err)
	assert.Equal(t, commit.String(), commitHash)
}

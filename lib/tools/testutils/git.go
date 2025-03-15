package testutils

import (
	"testing"

	bfs "github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/stretchr/testify/require"
)

// NewTestRepo creates a new test git repository with a memory filesystem.
func NewTestRepo(t *testing.T, opts ...repo.GitRepoOption) repo.GitRepo {
	opts = append(opts, repo.WithFS(bfs.NewInMemoryFs()))
	repo, err := repo.NewGitRepo("", NewNoopLogger(), opts...)
	require.NoError(t, err, "failed to create repo")
	require.NoError(t, repo.Init(), "failed to init repo")

	return repo
}

func NewTestRepoWithFS(t *testing.T, path string, fs *bfs.BillyFs, opts ...repo.GitRepoOption) repo.GitRepo {
	opts = append(opts, repo.WithFS(fs))
	repo, err := repo.NewGitRepo(path, NewNoopLogger(), opts...)
	require.NoError(t, err, "failed to create repo")
	require.NoError(t, repo.Init(), "failed to init repo")

	return repo
}

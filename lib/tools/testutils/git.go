package testutils

import (
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// NewTestRepo creates a new test git repository with a memory filesystem.
func NewTestRepo(t *testing.T, opts ...repo.GitRepoOption) repo.GitRepo {
	opts = append(opts, repo.WithMemFS())
	repo := repo.NewGitRepo(NewNoopLogger(), opts...)
	require.NoError(t, repo.Init(""), "failed to init repo")

	return repo
}

// NewTestRepoWithFS creates a new test git repository with the given filesystem.
func NewTestRepoWithFS(t *testing.T, fs afero.Fs, path string) repo.GitRepo {
	repo := repo.NewGitRepo(NewNoopLogger(), repo.WithFS(fs))
	require.NoError(t, repo.Init(path), "failed to init repo")

	return repo
}

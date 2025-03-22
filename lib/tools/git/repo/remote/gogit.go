package remote

import (
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage"
)

// GoGitRemoteInteractor is a GitRemoteInteractor that uses go-git.
type GoGitRemoteInteractor struct{}

func (g GoGitRemoteInteractor) Clone(s storage.Storer, worktree billy.Filesystem, o *git.CloneOptions) (*git.Repository, error) {
	return git.Clone(s, worktree, o)
}

func (g GoGitRemoteInteractor) Fetch(repo *git.Repository, o *git.FetchOptions) error {
	return repo.Fetch(o)
}

func (g GoGitRemoteInteractor) Push(repo *git.Repository, o *git.PushOptions) error {
	return repo.Push(o)
}

func (g GoGitRemoteInteractor) Pull(repo *git.Repository, o *git.PullOptions) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	return wt.Pull(o)
}

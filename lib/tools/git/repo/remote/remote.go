package remote

import (
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure -pkg mocks -out mocks/remote.go . GitRemoteInteractor

// GitRemoteInteractor is an interface for interacting with a git remote repository.
type GitRemoteInteractor interface {
	// Clone clones a repository.
	Clone(s storage.Storer, worktree billy.Filesystem, o *git.CloneOptions) (*git.Repository, error)

	// Fetch fetches changes from a repository.
	Fetch(repo *git.Repository, o *git.FetchOptions) error

	// Push pushes changes to a repository.
	Push(repo *git.Repository, o *git.PushOptions) error

	// Pull pulls changes from a repository.
	Pull(repo *git.Repository, o *git.PullOptions) error
}

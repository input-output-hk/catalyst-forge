package repo

import (
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	bfs "github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
)

type GitRepoOption func(*GitRepo)

// WithAuth sets the authentication for the interacting with a remote repository.
func WithAuth(username, password string) GitRepoOption {
	return func(g *GitRepo) {
		g.auth = &http.BasicAuth{
			Username: username,
			Password: password,
		}
	}
}

// WithAuthor sets the author for all commits.
func WithAuthor(name, email string) GitRepoOption {
	return func(g *GitRepo) {
		g.commitAuthor = name
		g.commitEmail = email
	}
}

// WithGitRemoteInteractor sets the remote interactor for the repository.
func WithGitRemoteInteractor(remote remote.GitRemoteInteractor) GitRepoOption {
	return func(g *GitRepo) {
		g.remote = remote
	}
}

// WithFS sets the filesystem for the repository.
func WithFS(fs fs.Filesystem) GitRepoOption {
	return func(g *GitRepo) {
		if bg, ok := fs.(*bfs.BillyFs); ok {
			g.fs = bg
		} else {
			panic("must use billy filesystem for git filesystem")
		}
	}
}

// CloneOption is an option for cloning a repository.
type CloneOption func(*gg.CloneOptions)

// WithCloneDepth sets the depth of the clone.
func WithCloneDepth(depth int) CloneOption {
	return func(o *gg.CloneOptions) {
		o.Depth = depth
	}
}

// WithRef sets the reference name for the clone.
func WithRef(ref string) CloneOption {
	return func(o *gg.CloneOptions) {
		o.ReferenceName = plumbing.ReferenceName(ref)
	}
}

// FetchOption is an option for fetching a repository.
type FetchOption func(*gg.FetchOptions)

// WithFetchDepth sets the depth of the fetch.
func WithFetchDepth(depth int) FetchOption {
	return func(o *gg.FetchOptions) {
		o.Depth = depth
	}
}

// WithRemoteName sets the name of the remote to fetch.
func WithRemoteName(name string) FetchOption {
	return func(o *gg.FetchOptions) {
		o.RemoteName = name
	}
}

// WithRefSpec sets the reference specification for the fetch.
func WithRefSpec(refSpec string) FetchOption {
	return func(fo *gg.FetchOptions) {
		fo.RefSpecs = []config.RefSpec{config.RefSpec(refSpec)}
	}
}

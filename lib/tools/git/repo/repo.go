package repo

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	bfs "github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
)

const (
	DEFAULT_AUTHOR = "Catalyst Forge"
	DEFAULT_EMAIL  = "forge@projectcatalyst.io"
)

var (
	ErrTagNotFound = fmt.Errorf("tag not found")
)

// GitRepo is a high-level representation of a git repository.
type GitRepo struct {
	auth         *http.BasicAuth
	basePath     string
	commitAuthor string
	commitEmail  string
	fs           *bfs.BillyFs
	gfs          *bfs.BillyFs
	wfs          *bfs.BillyFs
	logger       *slog.Logger
	raw          *gg.Repository
	remote       remote.GitRemoteInteractor
	worktree     *gg.Worktree
}

// CheckoutBranch checks out a branch with the given name.
func (g *GitRepo) CheckoutBranch(branch string) error {
	return g.worktree.Checkout(&gg.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
}

// CheckoutCommit checks out a commit with the given hash.
func (g *GitRepo) CheckoutCommit(hash string) error {
	return g.worktree.Checkout(&gg.CheckoutOptions{
		Hash: plumbing.NewHash(hash),
	})
}

// CheckoutRef checks out a reference (commit, branch, or tag) with the given name.
func (g *GitRepo) CheckoutRef(reference string) error {
	// Initialize checkout options
	checkoutOpts := &gg.CheckoutOptions{
		Force: true,
	}

	// Try as a commit hash first
	hash := plumbing.NewHash(reference)
	if !hash.IsZero() {
		_, err := g.raw.CommitObject(hash)
		if err == nil {
			checkoutOpts.Hash = hash
			return g.worktree.Checkout(checkoutOpts)
		}
	}

	// Try as a local branch
	branchRefName := plumbing.NewBranchReferenceName(reference)
	_, err := g.raw.Reference(branchRefName, true)
	if err == nil {
		checkoutOpts.Branch = branchRefName
		return g.worktree.Checkout(checkoutOpts)
	}

	// Try as a tag
	tagRefName := plumbing.NewTagReferenceName(reference)
	tagRef, err := g.raw.Reference(tagRefName, true)
	if err == nil {
		tagObj, err := g.raw.TagObject(tagRef.Hash())
		if err == nil {
			commit, err := tagObj.Commit()
			if err == nil {
				checkoutOpts.Hash = commit.Hash
				return g.worktree.Checkout(checkoutOpts)
			}
		} else if strings.Contains(err.Error(), "not a tag object") {
			checkoutOpts.Hash = tagRef.Hash()
			return g.worktree.Checkout(checkoutOpts)
		}
	}

	return fmt.Errorf("reference not found: %s is not a valid commit hash, branch, or tag", reference)
}

// Clone loads a repository from a git remote.
func (g *GitRepo) Clone(url string, opts ...CloneOption) error {
	g.logger.Debug("Cloning repository", "url", url)
	storage := filesystem.NewStorage(g.gfs.Raw(), cache.NewObjectLRUDefault())

	finalOpts := &gg.CloneOptions{
		URL:  url,
		Auth: g.auth,
	}
	for _, opt := range opts {
		opt(finalOpts)
	}
	repo, err := g.remote.Clone(storage, g.wfs.Raw(), finalOpts)

	if err != nil {
		return err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	g.raw = repo
	g.worktree = wt

	return nil
}

// Commit creates a commit with the given message.
func (g *GitRepo) Commit(msg string) (plumbing.Hash, error) {
	author, email := g.getAuthor()
	hash, err := g.worktree.Commit(msg, &gg.CommitOptions{
		Author: &object.Signature{
			Name:  author,
			Email: email,
			When:  time.Now(),
		},
	})

	if err != nil {
		return plumbing.ZeroHash, err
	}

	return hash, nil
}

// CreateTag creates a tag with the given name and message.
func (g *GitRepo) CreateTag(name, hash, message string) error {
	commitHash := plumbing.NewHash(hash)
	if commitHash.IsZero() {
		return fmt.Errorf("invalid commit hash: %s", hash)
	}

	_, err := g.raw.CommitObject(commitHash)
	if err != nil {
		return fmt.Errorf("failed to find commit %s: %w", hash, err)
	}

	author, email := g.getAuthor()
	opts := &gg.CreateTagOptions{
		Message: message,
		Tagger: &object.Signature{
			Name:  author,
			Email: email,
			When:  time.Now(),
		},
	}

	_, err = g.raw.CreateTag(name, commitHash, opts)
	if err != nil {
		return fmt.Errorf("failed to create tag %s: %w", name, err)
	}

	return nil
}

// Exists checks if a file exists in the repository.
func (g *GitRepo) Exists(path string) (bool, error) {
	_, err := g.wfs.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check if file exists: %w", err)
	}

	return true, nil
}

// Fetch fetches changes from the remote repository.
func (g *GitRepo) Fetch(opts ...FetchOption) error {
	g.logger.Debug("Fetching repository")
	fo := &gg.FetchOptions{
		Auth:       g.auth,
		RemoteName: "origin",
	}

	for _, opt := range opts {
		opt(fo)
	}

	err := g.remote.Fetch(g.raw, fo)

	if err != nil {
		return err
	}

	return nil
}

// GetCommit returns the commit with the given hash.
func (g *GitRepo) GetCommit(hash plumbing.Hash) (*object.Commit, error) {
	return g.raw.CommitObject(hash)
}

// GetCurrentBranch returns the name of the current branch.
func (g *GitRepo) GetCurrentBranch() (string, error) {
	head, err := g.raw.Head()
	if err != nil {
		return "", err
	}

	return head.Name().Short(), nil
}

// GetCurrentTag returns the tag of the current HEAD commit, if it exists.
func (g *GitRepo) GetCurrentTag() (string, error) {
	tags, err := g.raw.Tags()
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}

	ref, err := g.raw.Head()
	if err != nil {
		return "", err
	}

	var tag string
	err = tags.ForEach(func(t *plumbing.Reference) error {
		// Only process annotated tags
		tobj, err := g.raw.TagObject(t.Hash())
		if err != nil {
			return nil
		}

		if tobj.Target == ref.Hash() {
			tag = tobj.Name
			return nil
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to iterate over tags: %w", err)
	}

	if tag == "" {
		return "", ErrTagNotFound
	}

	return tag, nil
}

// GitFs returns the filesystem for the .git directory.
func (g *GitRepo) GitFs() fs.Filesystem {
	return g.gfs
}

// HasBranch returns true if the repository has a branch with the given name.
func (g *GitRepo) HasBranch(name string) (bool, error) {
	_, err := g.raw.Reference(plumbing.NewBranchReferenceName(name), false)
	if err != nil {
		return false, nil
	}

	return true, nil
}

// HasChanges returns true if the repository has changes.
func (g *GitRepo) HasChanges() (bool, error) {
	status, err := g.worktree.Status()
	if err != nil {
		return false, err
	}

	return !status.IsClean(), nil
}

// Head returns the HEAD reference of the current branch.
func (g *GitRepo) Head() (*plumbing.Reference, error) {
	return g.raw.Head()
}

// Init initializes a new repository at the given path.
func (g *GitRepo) Init() error {
	storage := filesystem.NewStorage(g.gfs.Raw(), cache.NewObjectLRUDefault())
	repo, err := gg.Init(storage, g.wfs.Raw())
	if err != nil {
		return fmt.Errorf("failed to init repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	g.raw = repo
	g.worktree = wt

	return nil
}

// IsErrNoUpdates returns true if the error is NoErrAlreadyUpToDate.
func IsErrNoUpdates(err error) bool {
	return errors.Is(err, gg.NoErrAlreadyUpToDate)
}

// MkdirAll creates a directory and all necessary parents.
func (g *GitRepo) MkdirAll(path string) error {
	return g.wfs.MkdirAll(filepath.Join(g.basePath, path), 0755)
}

// NewBranch creates a new branch with the given name.
func (g *GitRepo) NewBranch(name string) error {
	head, err := g.raw.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	return g.worktree.Checkout(&gg.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(name),
		Hash:   head.Hash(),
		Create: true,
	})
}

// NewTag creates a new tag with the given name and message.
func (g *GitRepo) NewTag(commit plumbing.Hash, name, message string) (*plumbing.Reference, error) {
	author, email := g.getAuthor()
	tag, err := g.raw.CreateTag(name, commit, &gg.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  author,
			Email: email,
			When:  time.Now(),
		},
		Message: message,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	return tag, nil
}

// Open loads a repository from a local path.
func (g *GitRepo) Open() error {
	storage := filesystem.NewStorage(g.gfs.Raw(), cache.NewObjectLRUDefault())
	repo, err := gg.Open(storage, g.wfs.Raw())
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	g.raw = repo
	g.worktree = wt

	return nil
}

// Pull fetches changes from the remote repository and merges them into the current branch.
func (g *GitRepo) Pull() error {
	return g.remote.Pull(g.raw, &gg.PullOptions{
		Auth:       g.auth,
		RemoteName: "origin",
	})
}

// Push pushes changes to the remote repository.
func (g *GitRepo) Push() error {
	return g.remote.Push(g.raw, &gg.PushOptions{
		Auth: g.auth,
	})
}

// Raw returns the underlying go-git repository.
func (g *GitRepo) Raw() *gg.Repository {
	return g.raw
}

// ReadFile reads the contents of a file in the repository.
func (g *GitRepo) ReadFile(path string) ([]byte, error) {
	return g.wfs.ReadFile(path)
}

// ReadDir reads the contents of a directory in the repository.
func (g *GitRepo) ReadDir(path string) ([]os.FileInfo, error) {
	return g.wfs.ReadDir(path)
}

// RemoveFile removes a file from the repository.
func (g *GitRepo) RemoveFile(path string) error {
	return g.wfs.Remove(path)
}

// SetAuth sets the authentication for the interacting with a remote repository.
func (g *GitRepo) SetAuth(auth *http.BasicAuth) {
	g.auth = auth
}

// StageFile adds a file to the staging area.
func (g *GitRepo) StageFile(path string) error {
	_, err := g.worktree.Add(path)
	if err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	return nil
}

// UnstageFile removes a file from the staging area.
func (g *GitRepo) UnstageFile(path string) error {
	_, err := g.worktree.Remove(path)
	if err != nil {
		return fmt.Errorf("failed to unstage file: %w", err)
	}

	return nil
}

// WorkFs returns the filesystem for the working directory.
func (g *GitRepo) WorkFs() fs.Filesystem {
	return g.wfs
}

// WriteFile writes the given contents to the given path in the repository.
// It also automatically adds the file to the staging area.
func (g *GitRepo) WriteFile(path string, contents []byte) error {
	if err := g.wfs.WriteFile(path, contents, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	_, err := g.worktree.Add(path)
	if err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}

	return nil
}

// getAuthor returns the author and email for commits.
func (g *GitRepo) getAuthor() (string, string) {
	author := g.commitAuthor
	email := g.commitEmail
	if author == "" {
		author = DEFAULT_AUTHOR
	}

	if email == "" {
		email = DEFAULT_EMAIL
	}

	return author, email
}

// NewGitRepo creates a new GitRepo instance.
func NewGitRepo(
	path string,
	logger *slog.Logger,
	opts ...GitRepoOption,
) (GitRepo, error) {
	r := GitRepo{
		logger: logger,
		remote: remote.GoGitRemoteInteractor{},
	}

	for _, opt := range opts {
		opt(&r)
	}

	if r.fs != nil {
		ng, err := r.fs.Raw().Chroot(filepath.Join(path, ".git"))
		if err != nil {
			return GitRepo{}, fmt.Errorf("failed to chroot git filesystem: %w", err)
		}

		nw, err := r.fs.Raw().Chroot(path)
		if err != nil {
			return GitRepo{}, fmt.Errorf("failed to chroot worktree filesystem: %w", err)
		}

		r.gfs = bfs.NewFs(ng)
		r.wfs = bfs.NewFs(nw)
	} else {
		r.gfs = bfs.NewOsFs(filepath.Join(path, ".git"))
		r.wfs = bfs.NewOsFs(path)
	}

	return r, nil
}

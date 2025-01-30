package repo

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
	"github.com/spf13/afero"
	df "gopkg.in/jfontan/go-billy-desfacer.v0"
)

const (
	DEFAULT_AUTHOR = "Catalyst Forge"
	DEFAULT_EMAIL  = "forge@projectcatalyst.io"
)

var (
	ErrTagNotFound = fmt.Errorf("tag not found")
)

type GitRepoOption func(*GitRepo)

// GitRepo is a high-level representation of a git repository.
type GitRepo struct {
	auth         *http.BasicAuth
	basePath     string
	commitAuthor string
	commitEmail  string
	fs           afero.Fs
	logger       *slog.Logger
	raw          *gg.Repository
	remote       remote.GitRemoteInteractor
	worktree     *gg.Worktree
}

// Clone loads a repository from a git remote.
func (g *GitRepo) Clone(path, url, branch string) error {
	workdir := afero.NewBasePathFs(g.fs, path)
	gitdir := afero.NewBasePathFs(g.fs, filepath.Join(path, ".git"))
	ref := fmt.Sprintf("refs/heads/%s", branch)

	g.logger.Debug("Cloning repository", "url", url, "ref", ref)
	storage := filesystem.NewStorage(df.New(gitdir), cache.NewObjectLRUDefault())
	repo, err := g.remote.Clone(storage, df.New(workdir), &gg.CloneOptions{
		URL:           url,
		Depth:         1,
		ReferenceName: plumbing.ReferenceName(ref),
		Auth:          g.auth,
	})

	if err != nil {
		return err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	g.basePath = path
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

// Exists checks if a file exists in the repository.
func (g *GitRepo) Exists(path string) (bool, error) {
	_, err := g.fs.Stat(filepath.Join(g.basePath, path))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check if file exists: %w", err)
	}

	return true, nil
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
func (g *GitRepo) Init(path string) error {
	var workdir afero.Fs
	if path == "" {
		workdir = g.fs
	} else {
		workdir = afero.NewBasePathFs(g.fs, path)
	}

	gitdir := afero.NewBasePathFs(g.fs, filepath.Join(path, ".git"))

	storage := filesystem.NewStorage(df.New(gitdir), cache.NewObjectLRUDefault())
	repo, err := gg.Init(storage, df.New(workdir))
	if err != nil {
		return fmt.Errorf("failed to init repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	g.basePath = path
	g.raw = repo
	g.worktree = wt

	return nil
}

// MkdirAll creates a directory and all necessary parents.
func (g *GitRepo) MkdirAll(path string) error {
	return g.fs.MkdirAll(filepath.Join(g.basePath, path), 0755)
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
func (g *GitRepo) Open(path string) error {
	workdir := afero.NewBasePathFs(g.fs, path)
	gitdir := afero.NewBasePathFs(g.fs, filepath.Join(path, ".git"))

	storage := filesystem.NewStorage(df.New(gitdir), cache.NewObjectLRUDefault())
	repo, err := gg.Open(storage, df.New(workdir))
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	g.basePath = path
	g.raw = repo
	g.worktree = wt

	return nil
}

// ReadFile reads the contents of a file in the repository.
func (g *GitRepo) ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(g.fs, path)
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

// ReadDir reads the contents of a directory in the repository.
func (g *GitRepo) ReadDir(path string) ([]os.FileInfo, error) {
	return afero.ReadDir(g.fs, filepath.Join(g.basePath, path))
}

// RemoveFile removes a file from the repository.
func (g *GitRepo) RemoveFile(path string) error {
	return g.fs.Remove(filepath.Join(g.basePath, path))
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

// WriteFile writes the given contents to the given path in the repository.
// It also automatically adds the file to the staging area.
func (g *GitRepo) WriteFile(path string, contents []byte) error {
	newPath := filepath.Join(g.basePath, path)
	if err := afero.WriteFile(g.fs, newPath, contents, 0644); err != nil {
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
func WithFS(fs afero.Fs) GitRepoOption {
	return func(g *GitRepo) {
		g.fs = fs
	}
}

// WithMemFS sets the repository to use an in-memory filesystem.
func WithMemFS() GitRepoOption {
	return func(g *GitRepo) {
		g.fs = afero.NewMemMapFs()
	}
}

// NewGitRepo creates a new GitRepo instance.
func NewGitRepo(logger *slog.Logger, opts ...GitRepoOption) GitRepo {
	r := GitRepo{
		logger: logger,
	}

	for _, opt := range opts {
		opt(&r)
	}

	if r.fs == nil {
		r.fs = afero.NewOsFs()
	} else if r.remote == nil {
		r.remote = remote.GoGitRemoteInteractor{}
	}

	return r
}

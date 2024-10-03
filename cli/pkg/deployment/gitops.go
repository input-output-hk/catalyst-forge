package deployment

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/project"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/secrets"
)

const GIT_NAME = "Catalyst Forge"
const GIT_EMAIL = "forge@projectcatalyst.io"
const GIT_MESSAGE = "chore: automatic deployment for %s"

var ErrNoChanges = fmt.Errorf("no changes to commit")

// gitRemoteInterface is an interface for interacting with a git remote.
type gitRemoteInterface interface {
	Clone(s storage.Storer, worktree billy.Filesystem, o *git.CloneOptions) (*git.Repository, error)
	Push(repo *git.Repository, o *git.PushOptions) error
}

// gitRemote is a concrete implementation of gitRemoteInterface.
type gitRemote struct{}

func (g gitRemote) Clone(s storage.Storer, worktree billy.Filesystem, o *git.CloneOptions) (*git.Repository, error) {
	return git.Clone(s, worktree, o)
}

func (g gitRemote) Push(repo *git.Repository, o *git.PushOptions) error {
	return repo.Push(o)
}

// GitopsDeployer is a deployer that deploys projects to a GitOps repository.
type GitopsDeployer struct {
	fs          billy.Filesystem
	repo        *git.Repository
	logger      *slog.Logger
	project     *project.Project
	remote      gitRemoteInterface
	secretStore *secrets.SecretStore
	token       string
	worktree    *git.Worktree
}

func (g *GitopsDeployer) Deploy() error {
	if (g.repo == nil) || (g.worktree == nil) {
		return fmt.Errorf("must load repository before calling Deploy")
	}

	globalDeploy := g.project.Blueprint.Global.Deployment
	prjDeploy := g.project.Blueprint.Project.Deployment
	envPath := filepath.Join(globalDeploy.Root, prjDeploy.Environment, "apps")
	prjPath := filepath.Join(envPath, g.project.Name)
	bundlePath := filepath.Join(prjPath, "bundle.cue")

	g.logger.Info("Checking if environment path exists", "path", envPath)
	exists, err := fileExists(g.fs, envPath)
	if err != nil {
		return fmt.Errorf("could not check if path exists: %w", err)
	} else if !exists {
		return fmt.Errorf("environment path does not exist: %s", envPath)
	}

	g.logger.Info("Checking if project path exists", "path", prjPath)
	exists, err = fileExists(g.fs, prjPath)
	if err != nil {
		return fmt.Errorf("could not check if path exists: %w", err)
	} else if !exists {
		g.logger.Info("Creating project path", "path", prjPath)
		err = g.fs.MkdirAll(prjPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("could not create project path: %w", err)
		}
	}

	g.logger.Info("Generating bundle")
	bundle, err := g.makeBundle()
	if err != nil {
		return fmt.Errorf("could not make bundle: %w", err)
	}

	g.logger.Info("Writing bundle to filesystem", "path", bundlePath)
	exists, err = fileExists(g.fs, bundlePath)
	if err != nil {
		return fmt.Errorf("could not check if bundle exists: %w", err)
	}

	var bundleFile billy.File
	if exists {
		bundleFile, err = g.fs.OpenFile(bundlePath, os.O_RDWR, os.ModePerm)
		if err != nil {
			return fmt.Errorf("could not open bundle file: %w", err)
		}
	} else {
		bundleFile, err = g.fs.Create(bundlePath)
		if err != nil {
			return fmt.Errorf("could not create bundle file: %w", err)
		}
	}

	_, err = bundleFile.Write(bundle)
	if err != nil {
		return fmt.Errorf("could not write bundle to file: %w", err)
	}

	if err := g.addFile(bundlePath); err != nil {
		return fmt.Errorf("could not add file to worktree: %w", err)
	}

	changes, err := g.hasChanges()
	if err != nil {
		return fmt.Errorf("could not check if worktree has changes: %w", err)
	} else if !changes {
		return ErrNoChanges
	}

	g.logger.Info("Committing changes", "path", bundlePath)
	if err := g.commit(); err != nil {
		return fmt.Errorf("could not commit changes: %w", err)
	}

	g.logger.Info("Pushing changes")
	if err := g.push(); err != nil {
		return fmt.Errorf("could not push changes: %w", err)
	}

	return nil
}

// Load loads the repository for the project.
func (g *GitopsDeployer) Load() error {
	var err error
	url := g.project.Blueprint.Global.Deployment.Repo.Url
	ref := g.project.Blueprint.Global.Deployment.Repo.Ref

	g.token, err = GetGitToken(g.project, g.secretStore, g.logger)
	if err != nil {
		return fmt.Errorf("could not get git provider token: %w", err)
	}

	if err := g.clone(url, ref); err != nil {
		return fmt.Errorf("could not clone repository: %w", err)
	}

	g.worktree, err = g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("could not get repository worktree: %w", err)
	}

	return nil
}

// addFile adds a file to the current worktree.
func (g *GitopsDeployer) addFile(path string) error {
	_, err := g.worktree.Add(path)
	if err != nil {
		return err
	}

	return nil
}

// clone clones the repository at the given URL and ref.
func (g *GitopsDeployer) clone(url, ref string) error {
	var err error
	ref = fmt.Sprintf("refs/heads/%s", ref)

	g.logger.Debug("Cloning repository", "url", url, "ref", ref)
	g.repo, err = g.remote.Clone(memory.NewStorage(), g.fs, &git.CloneOptions{
		URL:           url,
		Depth:         1,
		ReferenceName: plumbing.ReferenceName(ref),
		Auth: &http.BasicAuth{
			Username: "forge", // Note: this field is not used, but it cannot be empty.
			Password: g.token,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// commit commits all changes in the current worktree.
func (g *GitopsDeployer) commit() error {
	msg := fmt.Sprintf(GIT_MESSAGE, g.project.Name)
	_, err := g.worktree.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  GIT_NAME,
			Email: GIT_EMAIL,
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// hasChanges returns true if the current worktree has changes.
func (g *GitopsDeployer) hasChanges() (bool, error) {
	status, err := g.worktree.Status()
	if err != nil {
		return false, err
	}

	return !status.IsClean(), nil
}

// makeBundle generates a bundle file for the project.
func (g *GitopsDeployer) makeBundle() ([]byte, error) {
	bundle, err := GenerateBundle(g.project)
	if err != nil {
		return nil, fmt.Errorf("could not generate bundle: %w", err)
	}

	bytes, err := bundle.Encode()
	if err != nil {
		return nil, fmt.Errorf("could not encode bundle: %w", err)
	}

	return bytes, nil
}

// push pushes the current worktree to the remote repository.
func (g *GitopsDeployer) push() error {
	return g.remote.Push(g.repo, &git.PushOptions{
		Auth: &http.BasicAuth{
			Username: "forge", // Note: this field is not used, but it cannot be empty.
			Password: g.token,
		},
	})
}

// NewGitopsDeployer creates a new GitopsDeployer.
func NewGitopsDeployer(
	project *project.Project,
	store *secrets.SecretStore,
	logger *slog.Logger,
) GitopsDeployer {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return GitopsDeployer{
		fs:          memfs.New(),
		logger:      logger,
		project:     project,
		remote:      gitRemote{},
		secretStore: store,
	}
}

// fileExists checks if a file exists in the given filesystem.
func fileExists(fs billy.Filesystem, path string) (bool, error) {
	_, err := fs.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, fmt.Errorf("could not stat path: %w", err)
		}
	}

	return true, nil
}

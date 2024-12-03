package git

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
)

var (
	ErrGitRootNotFound = errors.New("git root not found")
)

// gitCheckoutOptions contains options for the Git checkout operation.
type gitCheckoutOptions struct {
	create     bool
	forceClean bool
}

// GitCheckoutOption is a helper type for setting Git checkout options.
type GitCheckoutOption func(*gitCheckoutOptions)

// GitCheckoutCreate sets the create option for the Git checkout.
// If the branch does not exist, it will be created.
// If the branch exists, it will be checked out.
func GitCheckoutCreate() GitCheckoutOption {
	return func(o *gitCheckoutOptions) {
		o.create = true
	}
}

// GitCheckoutForceClean sets the force clean option for the Git checkout.
func GitCheckoutForceClean() GitCheckoutOption {
	return func(o *gitCheckoutOptions) {
		o.forceClean = true
	}
}

// BranchExists checks if the given branch exists in the given Git repository.
func BranchExists(r *git.Repository, branch string) (bool, error) {
	branchRef := plumbing.NewBranchReferenceName(branch)

	_, err := r.Reference(plumbing.ReferenceName(branchRef), true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// CheckoutBranch checks out the given branch in the given Git repository.
func CheckoutBranch(r *git.Repository, branch string, opts ...GitCheckoutOption) error {
	var options gitCheckoutOptions
	for _, opt := range opts {
		opt(&options)
	}

	wt, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get git worktree: %w", err)
	}

	if options.forceClean {
		status, err := wt.Status()
		if err != nil {
			return fmt.Errorf("failed to get git worktree status: %w", err)
		}

		if !status.IsClean() {
			return fmt.Errorf("refusing to proceed due to dirty git worktree")
		}
	}

	var create bool
	if options.create {
		exists, err := BranchExists(r, branch)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}

		create = !exists
	} else {
		create = false
	}

	if err := wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
		Create: create,
	}); err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	return nil
}

// FindGitRoot finds the root of a Git repository starting from the given
// path. It returns the path to the root of the Git repository or an error if
// the root is not found.
func FindGitRoot(startPath string, rw walker.ReverseWalker) (string, error) {
	var gitRoot string
	err := rw.Walk(
		startPath,
		"/",
		func(path string, fileType walker.FileType, openFile func() (walker.FileSeeker, error)) error {
			if fileType == walker.FileTypeDir {
				if filepath.Base(path) == ".git" {
					gitRoot = filepath.Dir(path)
					return io.EOF
				}
			}

			return nil
		},
	)

	if err != nil {
		return "", err
	}

	if gitRoot == "" {
		return "", ErrGitRootNotFound
	}

	return gitRoot, nil
}

// GetCurrentBranch returns the name of the current branch in the given Git
func GetCurrentBranch(r *git.Repository) (string, error) {
	head, err := r.Head()
	if err != nil {
		return "", err
	}

	if !head.Name().IsBranch() {
		return "", errors.New("HEAD is not a branch")
	}

	return head.Name().Short(), nil
}

// InCI returns true if the code is running in a CI environment.
func InCI() bool {
	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		return true
	}

	return false
}

package handlers

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
)

type RepoHandler struct {
	deploymentFs   *billy.BillyFs
	deploymentRepo repo.GitRepo
	logger         *slog.Logger
	remote         remote.GitRemoteInteractor
	sourceFs       *billy.BillyFs
	sourceRepo     repo.GitRepo
	token          string
}

// LoadDeploymentRepo loads the deployment repository from the given URL and ref.
func (r *RepoHandler) LoadDeploymentRepo(url, ref string) error {
	rp, err := repo.NewCachedRepo(
		url,
		r.logger,
		repo.WithFS(r.deploymentFs),
		repo.WithGitRemoteInteractor(r.remote),
		repo.WithAuth("forge", r.token),
	)
	if err != nil {
		return err
	}

	if err := rp.CheckoutBranch(ref); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", ref, err)
	}

	if err := rp.Pull(); err != nil && !repo.IsErrNoUpdates(err) {
		return fmt.Errorf("failed to pull latest changes from branch %s: %w", ref, err)
	}

	r.deploymentRepo = rp

	return nil
}

// LoadSourceRepo loads the source repository from the given URL and commit.
func (r *RepoHandler) LoadSourceRepo(url, commit string) error {
	rp, err := repo.NewCachedRepo(
		url,
		r.logger,
		repo.WithFS(r.sourceFs),
		repo.WithGitRemoteInteractor(r.remote),
		repo.WithAuth("forge", r.token),
	)
	if err != nil {
		return err
	}

	if err := rp.Fetch(); err != nil && !repo.IsErrNoUpdates(err) {
		return fmt.Errorf("failed to fetch latest changes: %w", err)
	}

	if err := rp.CheckoutCommit(commit); err != nil {
		return fmt.Errorf("failed to checkout commit %s: %w", commit, err)
	}

	r.sourceRepo = rp

	return nil
}

// DeploymentRepo returns the deployment repository.
func (r *RepoHandler) DeploymentRepo() *repo.GitRepo {
	return &r.deploymentRepo
}

// SourceRepo returns the source repository.
func (r *RepoHandler) SourceRepo() *repo.GitRepo {
	return &r.sourceRepo
}

// NewRepoHandler creates a new RepoHandler.
func NewRepoHandler(
	deploymentFs *billy.BillyFs,
	sourceFs *billy.BillyFs,
	logger *slog.Logger,
	remote remote.GitRemoteInteractor,
	token string,
) *RepoHandler {
	if deploymentFs == nil {
		deploymentFs = billy.NewInMemoryFs()
	}

	if sourceFs == nil {
		sourceFs = billy.NewBaseOsFS()
	}

	return &RepoHandler{
		deploymentFs: deploymentFs,
		sourceFs:     sourceFs,
		logger:       logger,
		remote:       remote,
		token:        token,
	}
}

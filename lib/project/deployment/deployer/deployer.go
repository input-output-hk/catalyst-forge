package deployer

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/generator"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/providers"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
	"github.com/spf13/afero"
)

const (
	// This is the hardcoded default environment for the deployment.
	// We always deploy to the dev environment by default and do not allow
	// the user to change this.
	DEFAULT_ENV = "dev"

	// These are the hardcoded values to use when committing changes to the GitOps repository.
	GIT_NAME    = "Catalyst Forge"
	GIT_EMAIL   = "forge@projectcatalyst.io"
	GIT_MESSAGE = "chore: automatic deployment for %s"

	// This is the path to the project in the GitOps repository.
	// {root_path}/{environment}/{project_name}
	PATH = "%s/%s/%s"
)

var (
	ErrNoChanges = fmt.Errorf("no changes to commit")
)

// Deployer performs GitOps deployments for projects.
type Deployer struct {
	dryrun  bool
	fs      afero.Fs
	gen     generator.Generator
	logger  *slog.Logger
	project *project.Project
	remote  remote.GitRemoteInteractor
}

// DeployProject deploys the manifests for a project to the GitOps repository.
func (d *Deployer) Deploy() error {
	r, err := d.clone()
	if err != nil {
		return err
	}

	prjPath := d.buildProjectPath()

	d.logger.Info("Checking if project path exists", "path", prjPath)
	if err := d.checkProjectPath(prjPath, &r); err != nil {
		return fmt.Errorf("failed checking project path: %w", err)
	}

	d.logger.Info("Clearing project path", "path", prjPath)
	if err := d.clearProjectPath(prjPath, &r); err != nil {
		return fmt.Errorf("could not clear project path: %w", err)
	}

	d.logger.Info("Generating manifests")
	result, err := d.gen.GenerateBundle(d.project.Blueprint.Project.Deployment.Modules)
	if err != nil {
		return fmt.Errorf("could not generate deployment manifests: %w", err)
	}

	modPath := filepath.Join(prjPath, "mod.cue")
	d.logger.Info("Writing module", "path", modPath)
	if err := r.WriteFile(modPath, []byte(result.Module)); err != nil {
		return fmt.Errorf("could not write module: %w", err)
	}

	if err := r.StageFile(modPath); err != nil {
		return fmt.Errorf("could not add module to working tree: %w", err)
	}

	for name, result := range result.Manifests {
		manPath := filepath.Join(prjPath, fmt.Sprintf("%s.yaml", name))

		d.logger.Info("Writing manifest", "path", manPath)
		if err := r.WriteFile(manPath, []byte(result)); err != nil {
			return fmt.Errorf("could not write manifest: %w", err)
		}
		if err := r.StageFile(manPath); err != nil {
			return fmt.Errorf("could not add manifest to working tree: %w", err)
		}
	}

	if !d.dryrun {
		changes, err := r.HasChanges()
		if err != nil {
			return fmt.Errorf("could not check if worktree has changes: %w", err)
		} else if !changes {
			return ErrNoChanges
		}

		d.logger.Info("Committing changes")
		_, err = r.Commit(fmt.Sprintf(GIT_MESSAGE, d.project.Name))
		if err != nil {
			return fmt.Errorf("could not commit changes: %w", err)
		}

		d.logger.Info("Pushing changes")
		if err := r.Push(); err != nil {
			return fmt.Errorf("could not push changes: %w", err)
		}
	} else {
		d.logger.Info("Dry-run: not committing or pushing changes")
		d.logger.Info("Dumping manifests")
		for _, r := range result.Manifests {
			fmt.Println(string(r))
		}
	}

	return nil
}

// buildProjectPath builds the path to the project in the GitOps repository.
func (d *Deployer) buildProjectPath() string {
	globalDeploy := d.project.Blueprint.Global.Deployment
	return fmt.Sprintf(PATH, globalDeploy.Root, DEFAULT_ENV, d.project.Name)
}

// checkProjectPath checks if the project path exists and creates it if it does not.
func (d *Deployer) checkProjectPath(path string, r *repo.GitRepo) error {
	exists, err := r.Exists(path)
	if err != nil {
		return fmt.Errorf("could not check if project path exists: %w", err)
	} else if !exists {
		d.logger.Info("Creating project path", "path", path)
		err = r.MkdirAll(path)
		if err != nil {
			return fmt.Errorf("could not create project path: %w", err)
		}
	}

	return nil
}

// clearProjectPath clears the project path in the GitOps repository.
func (d *Deployer) clearProjectPath(path string, r *repo.GitRepo) error {
	files, err := r.ReadDir(path)
	if err != nil {
		return fmt.Errorf("could not read project path: %w", err)
	}

	for _, f := range files {
		path := filepath.Join(path, f.Name())
		d.logger.Debug("Removing file", "path", path)
		if err := r.RemoveFile(path); err != nil {
			return fmt.Errorf("could not remove file: %w", err)
		}

		if err := r.StageFile(path); err != nil {
			return fmt.Errorf("could not add file deletion to working tree: %w", err)
		}
	}

	return nil
}

// clone clones the GitOps repository.
func (d *Deployer) clone() (repo.GitRepo, error) {
	url := d.project.Blueprint.Global.Deployment.Repo.Url
	ref := d.project.Blueprint.Global.Deployment.Repo.Ref
	opts := []repo.GitRepoOption{
		repo.WithAuthor(GIT_NAME, GIT_EMAIL),
		repo.WithGitRemoteInteractor(d.remote),
		repo.WithFS(d.fs),
	}

	creds, err := providers.GetGitProviderCreds(d.project, d.logger)
	if err != nil {
		d.logger.Warn("could not get git provider credentials, not using any authentication", "error", err)
	} else {
		opts = append(opts, repo.WithAuth("forge", creds.Token))
	}

	d.logger.Info("Cloning repository", "url", url, "ref", ref)
	r := repo.NewGitRepo(d.logger, opts...)
	if err := r.Clone("/repo", url, ref); err != nil {
		return repo.GitRepo{}, fmt.Errorf("could not clone repository: %w", err)
	}

	return r, nil
}

// NewDeployer creates a new Deployer.
func NewDeployer(project *project.Project, store deployment.ManifestGeneratorStore, logger *slog.Logger, dryrun bool) Deployer {
	gen := generator.NewGenerator(store, logger)
	remote := remote.GoGitRemoteInteractor{}

	return Deployer{
		dryrun:  dryrun,
		gen:     gen,
		fs:      afero.NewMemMapFs(),
		logger:  logger,
		project: project,
		remote:  remote,
	}
}

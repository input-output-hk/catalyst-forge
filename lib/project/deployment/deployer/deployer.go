package deployer

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/generator"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/providers"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
	"github.com/spf13/afero"
)

const (
	// This is the hardcoded default environment for the deployment.
	// We always deploy to the dev environment by default and do not allow
	// the user to change this.
	// This is also the default for the `env` field in the deployment module.
	DEFAULT_ENV = "dev"

	// This is the name of the environment file which is merged with the deployment module.
	ENV_FILE = "env.mod.cue"

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

type DeployerConfig struct {
	Git         DeployerConfigGit
	RootDir     string
	ProjectName string
}

type DeployerConfigGit struct {
	Creds common.Secret
	Ref   string
	Url   string
}

// Deployer performs GitOps deployments for projects.
type Deployer struct {
	bundle deployment.ModuleBundle
	cfg    DeployerConfig
	ctx    *cue.Context
	dryrun bool
	fs     afero.Fs
	gen    generator.Generator
	logger *slog.Logger
	remote remote.GitRemoteInteractor
	ss     secrets.SecretStore
}

// DeployProject deploys the manifests for a project to the GitOps repository.
func (d *Deployer) Deploy() error {
	if d.bundle.Bundle.Env == "prod" {
		return fmt.Errorf("cannot deploy to production environment")
	}

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

	env, err := d.LoadEnv(prjPath, d.ctx, &r)
	if err != nil {
		return fmt.Errorf("could not load environment: %w", err)
	}

	d.logger.Info("Generating manifests")
	result, err := d.gen.GenerateBundle(d.bundle, env)
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
		_, err = r.Commit(fmt.Sprintf(GIT_MESSAGE, d.cfg.ProjectName))
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
	return fmt.Sprintf(PATH, d.cfg.RootDir, d.bundle.Bundle.Env, d.cfg.ProjectName)
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
		if f.Name() == ENV_FILE {
			continue
		}

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
	opts := []repo.GitRepoOption{
		repo.WithAuthor(GIT_NAME, GIT_EMAIL),
		repo.WithGitRemoteInteractor(d.remote),
		repo.WithFS(d.fs),
	}

	creds, err := providers.GetGitProviderCreds(&d.cfg.Git.Creds, &d.ss, d.logger)
	if err != nil {
		d.logger.Warn("could not get git provider credentials, not using any authentication", "error", err)
	} else {
		opts = append(opts, repo.WithAuth("forge", creds.Token))
	}

	d.logger.Info("Cloning repository", "url", d.cfg.Git.Url, "ref", d.cfg.Git.Ref)
	r := repo.NewGitRepo(d.logger, opts...)
	if err := r.Clone("/repo", d.cfg.Git.Url, d.cfg.Git.Ref); err != nil {
		return repo.GitRepo{}, fmt.Errorf("could not clone repository: %w", err)
	}

	return r, nil
}

// LoadEnv loads the environment file for the deployment, if it exists.
func (d *Deployer) LoadEnv(path string, ctx *cue.Context, r *repo.GitRepo) (cue.Value, error) {
	var env cue.Value

	envPath := filepath.Join(path, ENV_FILE)
	exists, err := r.Exists(envPath)
	if err != nil {
		return cue.Value{}, fmt.Errorf("could not check if environment file exists: %w", err)
	}

	if exists {
		d.logger.Info("Loading environment file", "path", envPath)
		contents, err := r.ReadFile(envPath)
		if err != nil {
			return cue.Value{}, fmt.Errorf("could not read environment file: %w", err)
		}

		env = ctx.CompileBytes(contents)
		if env.Err() != nil {
			return cue.Value{}, fmt.Errorf("could not compile environment file: %w", env.Err())
		}
	}

	return env, nil
}

// NewDeployer creates a new Deployer.
func NewDeployer(
	bundle deployment.ModuleBundle,
	cfg DeployerConfig,
	ms deployment.ManifestGeneratorStore,
	ss secrets.SecretStore,
	logger *slog.Logger,
	ctx *cue.Context,
	dryrun bool,
) Deployer {
	gen := generator.NewGenerator(ms, logger)
	remote := remote.GoGitRemoteInteractor{}

	return Deployer{
		bundle: bundle,
		cfg:    cfg,
		ctx:    ctx,
		dryrun: dryrun,
		gen:    gen,
		fs:     afero.NewMemMapFs(),
		logger: logger,
		remote: remote,
		ss:     ss,
	}
}

// NewDeployerConfigFromProject creates a DeployerConfig from a project.
func NewDeployerConfigFromProject(p *project.Project) DeployerConfig {
	return DeployerConfig{
		Git: DeployerConfigGit{
			Creds: p.Blueprint.Global.Ci.Providers.Git.Credentials,
			Ref:   p.Blueprint.Global.Deployment.Repo.Ref,
			Url:   p.Blueprint.Global.Deployment.Repo.Url,
		},
		RootDir:     p.Blueprint.Global.Deployment.Root,
		ProjectName: p.Name,
	}
}

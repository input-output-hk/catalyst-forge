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
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
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

// Deployment is a prepared deployment to a GitOps repository.
type Deployment struct {
	// Bundle is the deployment bundle being deployed.
	Bundle deployment.ModuleBundle

	// Manifests is the generated manifests for the deployment.
	// The key is the name of the manifest and the value is the manifest content.
	Manifests map[string][]byte

	// RawBundle is the raw representation of the deployment bundle (in CUE).
	RawBundle []byte

	// Repo is an in-memory clone of the GitOps repository being deployed to.
	Repo repo.GitRepo

	// Project is the name of the project being deployed.
	Project string

	logger *slog.Logger
}

// DeployerConfig is the configuration for a Deployer.
type DeployerConfig struct {
	// Git is the configuration for the GitOps repository.
	Git DeployerConfigGit

	// RootDir is the root directory in the GitOps repository to deploy to.
	RootDir string
}

// DeployerConfigGit is the configuration for the GitOps repository.
type DeployerConfigGit struct {
	// Creds is the credentials to use for the GitOps repository.
	Creds common.Secret

	// Ref is the Git reference to deploy to.
	Ref string

	// Url is the URL of the GitOps repository.
	Url string
}

// Deployer performs GitOps deployments for projects.
type Deployer struct {
	cfg    DeployerConfig
	ctx    *cue.Context
	fs     fs.Filesystem
	gen    generator.Generator
	logger *slog.Logger
	remote remote.GitRemoteInteractor
	ss     secrets.SecretStore
}

// CloneOptions are options for cloning a repository.
type CloneOptions struct {
	fs fs.Filesystem
}

// CloneOption is an option for cloning a repository.
type CloneOption func(*CloneOptions)

// WithFS sets the filesystem to use for cloning a repository.
func WithFS(fs fs.Filesystem) CloneOption {
	return func(o *CloneOptions) {
		o.fs = fs
	}
}

// CreateDeployment creates a deployment for the given project and bundle.
func (d *Deployer) CreateDeployment(
	project string,
	bundle deployment.ModuleBundle,
	opts ...CloneOption,
) (*Deployment, error) {
	options := CloneOptions{
		fs: billy.NewInMemoryFs(),
	}
	for _, o := range opts {
		o(&options)
	}

	r, err := d.clone(d.cfg.Git.Url, d.cfg.Git.Ref, options.fs)
	if err != nil {
		return nil, err
	}

	prjPath := buildProjectPath(d.cfg.RootDir, project, bundle)
	d.logger.Info("Checking if project path exists", "path", prjPath)
	if err := d.checkProjectPath(prjPath, &r); err != nil {
		return nil, fmt.Errorf("failed checking project path: %w", err)
	}

	env, err := d.LoadEnv(prjPath, d.ctx, &r)
	if err != nil {
		return nil, fmt.Errorf("could not load environment: %w", err)
	}

	d.logger.Info("Generating manifests")
	result, err := d.gen.GenerateBundle(bundle, env)
	if err != nil {
		return nil, fmt.Errorf("could not generate deployment manifests: %w", err)
	}

	d.logger.Info("Clearing project path", "path", prjPath)
	if err := d.clearProjectPath(prjPath, &r); err != nil {
		return nil, fmt.Errorf("could not clear project path: %w", err)
	}

	bundlePath := filepath.Join(prjPath, "bundle.cue")
	d.logger.Info("Writing bundle", "path", bundlePath)
	if err := r.WriteFile(bundlePath, []byte(result.Module)); err != nil {
		return nil, fmt.Errorf("could not write bundle: %w", err)
	}

	if err := r.StageFile(bundlePath); err != nil {
		return nil, fmt.Errorf("could not add bundle to working tree: %w", err)
	}

	for name, result := range result.Manifests {
		manPath := filepath.Join(prjPath, fmt.Sprintf("%s.yaml", name))

		d.logger.Info("Writing manifest", "path", manPath)
		if err := r.WriteFile(manPath, []byte(result)); err != nil {
			return nil, fmt.Errorf("could not write manifest: %w", err)
		}
		if err := r.StageFile(manPath); err != nil {
			return nil, fmt.Errorf("could not add manifest to working tree: %w", err)
		}
	}

	return &Deployment{
		Bundle:    bundle,
		Manifests: result.Manifests,
		RawBundle: result.Module,
		Repo:      r,
		logger:    d.logger,
	}, nil
}

func (d *Deployer) FetchBundle(url, ref, projectPath string, opts ...CloneOption) (deployment.ModuleBundle, error) {
	options := CloneOptions{
		fs: billy.NewInMemoryFs(),
	}
	for _, o := range opts {
		o(&options)
	}

	_, err := d.clone(url, ref, options.fs)
	if err != nil {
		return deployment.ModuleBundle{}, err
	}

	return deployment.ModuleBundle{}, nil
}

// Commit commits the deployment to the GitOps repository.
func (d *Deployment) Commit() error {
	d.logger.Info("Committing changes")
	_, err := d.Repo.Commit(fmt.Sprintf(GIT_MESSAGE, d.Project))
	if err != nil {
		return fmt.Errorf("could not commit changes: %w", err)
	}

	d.logger.Info("Pushing changes")
	if err := d.Repo.Push(); err != nil {
		return fmt.Errorf("could not push changes: %w", err)
	}

	return nil
}

// HasChanges checks if the deployment results in changes to the GitOps repository.
func (d *Deployment) HasChanges() (bool, error) {
	changes, err := d.Repo.HasChanges()
	if err != nil {
		return false, fmt.Errorf("could not check if worktree has changes: %w", err)
	}

	return changes, nil
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

// clone clones the given repository and returns the GitRepo.
func (d *Deployer) clone(url, ref string, fs fs.Filesystem) (repo.GitRepo, error) {
	opts := []repo.GitRepoOption{
		repo.WithAuthor(GIT_NAME, GIT_EMAIL),
		repo.WithGitRemoteInteractor(d.remote),
		repo.WithFS(fs),
	}

	creds, err := providers.GetGitProviderCreds(&d.cfg.Git.Creds, &d.ss, d.logger)
	if err != nil {
		d.logger.Warn("could not get git provider credentials, not using any authentication", "error", err)
	} else {
		opts = append(opts, repo.WithAuth("forge", creds.Token))
	}

	d.logger.Info("Cloning repository", "url", url, "ref", ref)
	r, err := repo.NewGitRepo("/repo", d.logger, opts...)
	if err != nil {
		return repo.GitRepo{}, fmt.Errorf("could not create git repository: %w", err)
	}

	if err := r.Clone(url, ref); err != nil {
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
	cfg DeployerConfig,
	ms deployment.ManifestGeneratorStore,
	ss secrets.SecretStore,
	logger *slog.Logger,
	ctx *cue.Context,
) Deployer {
	gen := generator.NewGenerator(ms, logger)
	remote := remote.GoGitRemoteInteractor{}

	return Deployer{
		cfg:    cfg,
		ctx:    ctx,
		gen:    gen,
		fs:     billy.NewBaseOsFS(),
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
		RootDir: p.Blueprint.Global.Deployment.Root,
	}
}

// buildProjectPath builds the path to the project in the GitOps repository.
func buildProjectPath(root string, project string, b deployment.ModuleBundle) string {
	return fmt.Sprintf(PATH, root, b.Bundle.Env, project)
}

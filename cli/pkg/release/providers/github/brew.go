package github

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"

	"log/slog"

	"github.com/Masterminds/sprig/v3"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	gp "github.com/input-output-hk/catalyst-forge/lib/providers/git"
	"github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
)

const (
	// DefaultTemplatesUrl is the default URL for the Homebrew templates.
	DefaultTemplatesUrl = "https://raw.githubusercontent.com/input-output-hk/catalyst-forge/master/templates/brew"
)

// BrewTemplateData represents the data that will be used to render the Homebrew template.
type BrewTemplateData struct {
	Name        string
	Description string
	Homepage    string
	Version     string
	BinaryName  string
	Assets      map[string]BrewAsset
}

// BrewAsset represents an asset that will be used to render the Homebrew template.
type BrewAsset struct {
	URL    string
	SHA256 string
}

// BrewDeployer handles the logic for deploying a Homebrew formula.
type BrewDeployer struct {
	cfg          *ReleaseConfig
	fs           fs.Filesystem
	gitFs        fs.Filesystem
	logger       *slog.Logger
	project      project.Project
	remote       remote.GitRemoteInteractor
	secretsStore secrets.SecretStore
	workdir      string
}

// Deploy generates and publishes the Homebrew formula.
func (d *BrewDeployer) Deploy(releaseName string, assets map[string]string) error {
	d.logger.Info("Starting Homebrew deployment")

	templateData, err := d.getTemplateData(assets)
	if err != nil {
		return fmt.Errorf("failed to get template data: %w", err)
	}

	renderedTemplate, err := d.renderTemplate(templateData)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	if err := d.publishToTap(renderedTemplate); err != nil {
		return fmt.Errorf("failed to publish to tap: %w", err)
	}

	d.logger.Info("Homebrew deployment successfully mocked.", "assets", fmt.Sprintf("%v", assets))
	return nil
}

// publishToTap publishes the rendered template to the tap repository.
func (d *BrewDeployer) publishToTap(content string) error {
	d.logger.Info("Publishing to tap repository", "repo", d.cfg.Brew.Tap.Repository)

	r, err := d.cloneTapRepo()
	if err != nil {
		return fmt.Errorf("failed to clone tap repository: %w", err)
	}

	recipePath := fmt.Sprintf("Formula/%s.rb", d.project.Name)
	if err := r.WriteFile(recipePath, []byte(content)); err != nil {
		return fmt.Errorf("failed to write recipe file: %w", err)
	}

	commitMsg := fmt.Sprintf("feat(brew): update %s to version %s", d.project.Name, d.project.Tag.Full)
	if _, err := r.Commit(commitMsg); err != nil {
		return fmt.Errorf("failed to commit recipe: %w", err)
	}
	if err := r.Push(); err != nil {
		return fmt.Errorf("failed to push recipe: %w", err)
	}

	d.logger.Info("Successfully published to tap repository")
	return nil
}

func (d *BrewDeployer) cloneTapRepo() (*repo.GitRepo, error) {
	creds, err := gp.GetGitProviderCreds(&d.project.Blueprint.Global.Ci.Providers.Git.Credentials, &d.secretsStore, d.logger)
	if err != nil {
		d.logger.Warn("could not get git provider credentials, not using any authentication", "error", err)
	}

	opts := []repo.GitRepoOption{
		repo.WithFS(d.gitFs),
		repo.WithGitRemoteInteractor(d.remote),
	}
	if creds.Token != "" {
		opts = append(opts, repo.WithAuth("forge", creds.Token))
	}

	r, err := repo.NewGitRepo("/repo", d.logger, opts...)
	if err != nil {
		return nil, fmt.Errorf("could not create git repository: %w", err)
	}

	if err := r.Clone(d.cfg.Brew.Tap.Repository, repo.WithRef(d.cfg.Brew.Tap.Branch), repo.WithCloneDepth(1)); err != nil {
		return nil, fmt.Errorf("could not clone repository: %w", err)
	}

	return &r, nil
}

// getTemplateData generates the template data for the Homebrew formula.
func (d *BrewDeployer) getTemplateData(assets map[string]string) (*BrewTemplateData, error) {
	data := &BrewTemplateData{
		Name:        d.project.Name,
		Description: d.cfg.Brew.Description,
		Homepage:    fmt.Sprintf("https://github.com/%s", d.project.Blueprint.Global.Repo.Name),
		Version:     d.project.Tag.Full,
		BinaryName:  d.cfg.Brew.BinaryName,
		Assets:      make(map[string]BrewAsset),
	}

	for platform, url := range assets {
		filename := fmt.Sprintf("%s-%s.tar.gz", d.cfg.Prefix, strings.Replace(platform, "/", "-", -1))
		path := filepath.Join(d.workdir, filename)

		sha, err := d.calculateSHA256(path)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate SHA256 for %s: %w", path, err)
		}

		assetKey := d.platformToAssetKey(platform)
		data.Assets[assetKey] = BrewAsset{
			URL:    url,
			SHA256: sha,
		}
	}

	return data, nil
}

// fetchTemplateFromGit fetches a template from a git repository.
func (d *BrewDeployer) fetchTemplateFromGit() ([]byte, error) {
	// Create a temporary filesystem for the template repo
	tempFs := billy.NewInMemoryFs()
	
	// Create a git repo with temporary filesystem
	templateRepo, err := repo.NewGitRepo(
		"/template-repo",
		d.logger,
		repo.WithFS(tempFs),
		repo.WithGitRemoteInteractor(d.remote),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create template git repository: %w", err)
	}

	// Clone the template repository
	cloneOpts := []repo.CloneOption{
		repo.WithCloneDepth(1),
	}
	if d.cfg.Brew.Templates.Branch != "" {
		cloneOpts = append(cloneOpts, repo.WithRef(d.cfg.Brew.Templates.Branch))
	}
	
	if err := templateRepo.Clone(d.cfg.Brew.Templates.Repository, cloneOpts...); err != nil {
		return nil, fmt.Errorf("could not clone template repository: %w", err)
	}

	// Read the template file
	templatePath := fmt.Sprintf("/template-repo/%s.rb.tpl", d.cfg.Brew.Template)
	templateContent, err := tempFs.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("could not read template file %s: %w", templatePath, err)
	}

	return templateContent, nil
}

// renderTemplate renders the Homebrew template.
func (d *BrewDeployer) renderTemplate(data *BrewTemplateData) (string, error) {
	var templateContent []byte
	var err error

	// Check if we should fetch from a git repository or HTTP URL
	if d.cfg.Brew.Templates != nil {
		// Fetch template from git repository
		templateContent, err = d.fetchTemplateFromGit()
		if err != nil {
			return "", fmt.Errorf("failed to fetch template from git: %w", err)
		}
	} else {
		// Fall back to HTTP URL (for backward compatibility)
		templatesUrl := DefaultTemplatesUrl
		if d.cfg.Brew.TemplatesUrl != "" {
			templatesUrl = d.cfg.Brew.TemplatesUrl
		}
		templateURL := fmt.Sprintf("%s/%s.rb.tpl", templatesUrl, d.cfg.Brew.Template)

		resp, err := http.Get(templateURL)
		if err != nil {
			return "", fmt.Errorf("failed to download template from %s: %w", templateURL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to download template: received status code %d", resp.StatusCode)
		}

		templateContent, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read template content: %w", err)
		}
	}

	tmpl, err := template.New("brew").Funcs(sprig.TxtFuncMap()).Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return rendered.String(), nil
}

// calculateSHA256 calculates the SHA256 hash of a file.
func (d *BrewDeployer) calculateSHA256(path string) (string, error) {
	file, err := d.fs.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// platformToAssetKey converts a platform to an asset key.
func (d *BrewDeployer) platformToAssetKey(platform string) string {
	switch platform {
	case "darwin/amd64":
		return "DarwinAMD64"
	case "darwin/arm64":
		return "DarwinARM64"
	case "linux/amd64":
		return "LinuxAMD64"
	case "linux/arm64":
		return "LinuxARM64"
	default:
		return ""
	}
}

// BrewDeployerOption is a function that can be used to configure the brew deployer.
type BrewDeployerOption func(*BrewDeployer)

// WithGitFilesystem sets the filesystem to use for the git repository.
func WithGitFilesystem(gitFs fs.Filesystem) BrewDeployerOption {
	return func(d *BrewDeployer) {
		d.gitFs = gitFs
	}
}

// WithLogger sets the logger to use for the brew deployer.
func WithLogger(logger *slog.Logger) BrewDeployerOption {
	return func(d *BrewDeployer) {
		d.logger = logger
	}
}

// WithFilesystem sets the filesystem to use for the brew deployer.
func WithFilesystem(fs fs.Filesystem) BrewDeployerOption {
	return func(d *BrewDeployer) {
		d.fs = fs
	}
}

// WithSecretsStore sets the secrets store to use for the brew deployer.
func WithSecretsStore(secretsStore secrets.SecretStore) BrewDeployerOption {
	return func(d *BrewDeployer) {
		d.secretsStore = secretsStore
	}
}

// WithRemote sets the git remote interactor to use for the brew deployer.
func WithRemote(remote remote.GitRemoteInteractor) BrewDeployerOption {
	return func(d *BrewDeployer) {
		d.remote = remote
	}
}

// WithProject sets the project to use for the brew deployer.
func WithProject(project project.Project) BrewDeployerOption {
	return func(d *BrewDeployer) {
		d.project = project
	}
}

// NewBrewDeployer creates a new instance of BrewDeployer.
func NewBrewDeployer(cfg *ReleaseConfig, workdir string, opts ...BrewDeployerOption) *BrewDeployer {
	d := &BrewDeployer{
		cfg:          cfg,
		fs:           billy.NewBaseOsFS(),
		secretsStore: secrets.NewDefaultSecretStore(),
		workdir:      workdir,
		gitFs:        billy.NewBaseOsFS(), // Use OS filesystem for git operations in production
		remote:       remote.GoGitRemoteInteractor{},
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

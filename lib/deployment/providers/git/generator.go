package git

import (
	"fmt"
	"log/slog"
	"strings"

	"cuelang.org/go/cue"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
)

// Options is the configuration for the GitManifestGenerator.
type Options struct {
	Paths []string `json:"paths"`
}

// GitManifestGenerator is a ManifestGenerator that uses Git.
// It clones the git repository and checks out the specified ref.
// It then reads the file at the specified path and returns its contents.
type GitManifestGenerator struct {
	fs     fs.Filesystem
	logger *slog.Logger
	remote remote.GitRemoteInteractor
}

func (g *GitManifestGenerator) Generate(mod sp.Module, raw cue.Value, env string) ([]byte, error) {
	var opts Options
	v := raw.LookupPath(cue.ParsePath("values"))
	if err := v.Decode(&opts); err != nil {
		return nil, fmt.Errorf("failed to decode options: %w", err)
	} else if len(opts.Paths) == 0 {
		return nil, fmt.Errorf("no paths specified")
	}

	r, err := repo.NewGitRepo("/repo", g.logger, repo.WithFS(g.fs), repo.WithGitRemoteInteractor(g.remote))
	if err != nil {
		return nil, fmt.Errorf("failed to create git repo: %w", err)
	}

	g.logger.Debug("Cloning git repo", "url", mod.Registry)
	if err := r.Clone(mod.Registry); err != nil {
		return nil, fmt.Errorf("failed to clone git repo %s: %w", mod.Registry, err)
	}

	g.logger.Debug("Checking out git ref", "ref", mod.Version)
	if err := r.CheckoutRef(mod.Version); err != nil {
		return nil, fmt.Errorf("failed to checkout ref %s: %w", mod.Version, err)
	}

	var final string
	for _, path := range opts.Paths {
		g.logger.Debug("Reading file from git repo", "path", path)
		exists, err := r.Exists(path)
		if err != nil {
			return nil, fmt.Errorf("failed to check if path %s exists: %w", path, err)
		} else if !exists {
			return nil, fmt.Errorf("path %s does not exist in git repo", path)
		}

		contents, err := r.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", path, err)
		}

		if final == "" {
			final = strings.TrimPrefix(string(contents), "---\n")
		} else {
			final += "\n---\n" + strings.TrimPrefix(string(contents), "---\n")
		}
	}

	return []byte(final), nil
}

// NewGitManifestGenerator creates a new GitManifestGenerator instance.
func NewGitManifestGenerator(logger *slog.Logger) *GitManifestGenerator {
	return &GitManifestGenerator{
		fs:     billy.NewInMemoryFs(),
		logger: logger,
		remote: remote.GoGitRemoteInteractor{},
	}
}

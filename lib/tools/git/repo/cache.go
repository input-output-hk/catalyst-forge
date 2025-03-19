package repo

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	bfs "github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
)

// NewCachedRepo creates a new GitRepo instance that uses a predefined cache path.
// If the repository does not exist in the cache, it will be cloned.
// If the repository exists in the cache, it will be opened.
func NewCachedRepo(url string, logger *slog.Logger, opts ...GitRepoOption) (GitRepo, error) {
	cUrl := strings.TrimPrefix(url, "https://")
	cUrl = strings.TrimPrefix(cUrl, "git://")
	cUrl = strings.TrimSuffix(cUrl, ".git")

	path := filepath.Join(xdg.CacheHome, "forge")
	gp := filepath.Join(path, cUrl, ".git")
	wp := filepath.Join(path, cUrl)

	r := GitRepo{
		logger: logger,
		remote: remote.GoGitRemoteInteractor{},
	}

	for _, opt := range opts {
		opt(&r)
	}

	if r.fs != nil {
		ng, err := r.fs.Raw().Chroot(gp)
		if err != nil {
			return GitRepo{}, fmt.Errorf("failed to chroot git filesystem: %w", err)
		}

		nw, err := r.fs.Raw().Chroot(wp)
		if err != nil {
			return GitRepo{}, fmt.Errorf("failed to chroot worktree filesystem: %w", err)
		}

		r.gfs = bfs.NewFs(ng)
		r.wfs = bfs.NewFs(nw)
	} else {
		r.fs = bfs.NewBaseOsFS()
		r.gfs = bfs.NewOsFs(gp)
		r.wfs = bfs.NewOsFs(wp)
	}

	exists, err := r.fs.Exists(filepath.Join(gp))
	if err != nil {
		return GitRepo{}, fmt.Errorf("could not check if repo exists: %w", err)
	} else if !exists {
		r.logger.Info("No cached repo found, cloning", "url", url)
		if err := r.Clone(url); err != nil {
			return GitRepo{}, fmt.Errorf("could not clone: %w", err)
		}
	} else {
		r.logger.Info("Cached repo found, opening", "url", url)
		if err := r.Open(); err != nil {
			return GitRepo{}, fmt.Errorf("could not open repo: %w", err)
		}
	}

	return r, nil
}

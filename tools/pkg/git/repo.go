package git

import (
	"fmt"
	"path/filepath"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/spf13/afero"
	df "gopkg.in/jfontan/go-billy-desfacer.v0"
)

type RepoLoader interface {
	Load(path string) (*gg.Repository, error)
}

type DefaultRepoLoader struct {
	fs afero.Fs
}

func (r *DefaultRepoLoader) Load(path string) (*gg.Repository, error) {
	return loadFromAfero(r.fs, path)
}

func NewDefaultRepoLoader() DefaultRepoLoader {
	return DefaultRepoLoader{
		fs: afero.NewOsFs(),
	}
}

func NewCustomDefaultRepoLoader(fs afero.Fs) DefaultRepoLoader {
	return DefaultRepoLoader{
		fs: fs,
	}
}

func loadFromAfero(fs afero.Fs, path string) (*gg.Repository, error) {
	workdir := afero.NewBasePathFs(fs, path)
	gitdir := afero.NewBasePathFs(fs, filepath.Join(path, ".git"))

	storage := filesystem.NewStorage(df.New(gitdir), cache.NewObjectLRUDefault())
	repo, err := gg.Open(storage, df.New(workdir))
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	return repo, nil
}

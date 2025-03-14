package main

import (
	"log"
	"path/filepath"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/spf13/afero"
	df "gopkg.in/jfontan/go-billy-desfacer.v0"
)

func main() {
	fs := afero.NewMemMapFs()
	workdir := afero.NewBasePathFs(fs, "/repo")
	gitdir := afero.NewBasePathFs(fs, filepath.Join("/repo", ".git"))

	storage := filesystem.NewStorage(df.New(gitdir), cache.NewObjectLRUDefault())
	_, err := gg.Clone(storage, df.New(workdir), &gg.CloneOptions{
		Depth:         1,
		ReferenceName: "refs/heads/main",
		URL:           "https://github.com/input-output-hk/catalyst-voices",
	})
	if err != nil {
		log.Fatalf("failed cloning repository: %v", err)
	}
}

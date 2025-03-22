package test

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/generator"
	dm "github.com/input-output-hk/catalyst-forge/lib/project/deployment/mocks"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
	rm "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
)

type RemoteOptions struct {
	Clone *gg.CloneOptions
	Push  *gg.PushOptions
}

func NewMockGenerator(manifest string) generator.Generator {
	return generator.NewGenerator(
		NewMockManifestStore(manifest),
		testutils.NewNoopLogger(),
	)
}

func NewMockManifestStore(manifest string) deployment.ManifestGeneratorStore {
	return deployment.NewManifestGeneratorStore(
		map[deployment.Provider]func(*slog.Logger) deployment.ManifestGenerator{
			deployment.ProviderKCL: func(logger *slog.Logger) deployment.ManifestGenerator {
				return &dm.ManifestGeneratorMock{
					GenerateFunc: func(mod sp.Module, env string) ([]byte, error) {
						return []byte(manifest), nil
					},
				}
			},
		},
	)
}

func NewMockGitRemoteInterface(
	files map[string]string,
) (remote.GitRemoteInteractor, RemoteOptions, error) {
	var cloneOpts gg.CloneOptions
	var pushOpts gg.PushOptions
	return &rm.GitRemoteInteractorMock{
			CloneFunc: func(s storage.Storer, worktree billy.Filesystem, o *gg.CloneOptions) (*gg.Repository, error) {
				cloneOpts = *o
				repo, err := gg.Init(s, worktree)
				if err != nil {
					return nil, fmt.Errorf("failed to init repo: %w", err)
				}

				wt, err := repo.Worktree()
				if err != nil {
					return nil, fmt.Errorf("failed to get worktree: %w", err)
				}

				if files != nil {
					for path, content := range files {
						dir := filepath.Dir(path)
						if err := worktree.MkdirAll(dir, 0755); err != nil {
							return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
						}

						f, err := worktree.Create(path)
						if err != nil {
							return nil, fmt.Errorf("failed to create file %s: %w", path, err)
						}
						_, err = f.Write([]byte(content))
						if err != nil {
							return nil, fmt.Errorf("failed to write to file %s: %w", path, err)
						}

						_, err = wt.Add(path)
						if err != nil {
							return nil, fmt.Errorf("failed to add file: %w", err)
						}
					}

					_, err = wt.Commit("initial commit", &gg.CommitOptions{
						Author: &object.Signature{
							Name:  "test",
							Email: "test@test.com",
						},
					})
					if err != nil {
						return nil, fmt.Errorf("failed to commit: %w", err)
					}
				}

				return repo, nil
			},
			PushFunc: func(repo *gg.Repository, o *gg.PushOptions) error {
				pushOpts = *o
				return nil
			},
		}, RemoteOptions{
			Clone: &cloneOpts,
			Push:  &pushOpts,
		}, nil
}

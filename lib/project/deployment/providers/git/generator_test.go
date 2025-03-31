package git

import (
	"fmt"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	bfs "github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
	rm "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitManifestGeneratorGenerate(t *testing.T) {
	ctx := cuecontext.New()
	tests := []struct {
		name     string
		mod      sp.Module
		files    map[string]string
		validate func(*testing.T, []byte, remoteOptions, error)
	}{
		{
			name: "single path",
			mod: sp.Module{
				Instance:  "instance",
				Name:      "name",
				Namespace: "default",
				Registry:  "https://github.com/owner/repo",
				Values:    ctx.CompileString(`{paths: ["project/file.yaml"]}`),
				Version:   "master",
			},
			files: map[string]string{
				"project/file.yaml": "file contents",
			},
			validate: func(t *testing.T, got []byte, opts remoteOptions, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "file contents", string(got))
				assert.Equal(t, "https://github.com/owner/repo", opts.Clone.URL)
			},
		},
		{
			name: "multiple paths",
			mod: sp.Module{
				Instance:  "instance",
				Name:      "name",
				Namespace: "default",
				Registry:  "https://github.com/owner/repo",
				Values:    ctx.CompileString(`{paths: ["project/file.yaml", "project1/file1.yaml"]}`),
				Version:   "master",
			},
			files: map[string]string{
				"project/file.yaml":   "file contents",
				"project1/file1.yaml": "file contents",
			},
			validate: func(t *testing.T, got []byte, opts remoteOptions, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "file contents\n---\nfile contents", string(got))
				assert.Equal(t, "https://github.com/owner/repo", opts.Clone.URL)
			},
		},
		{
			name: "no paths",
			mod: sp.Module{
				Instance:  "instance",
				Name:      "name",
				Namespace: "default",
				Registry:  "https://github.com/owner/repo",
				Values:    ctx.CompileString(`{}`),
				Version:   "master",
			},
			files: map[string]string{
				"project/file.yaml": "file contents",
			},
			validate: func(t *testing.T, got []byte, opts remoteOptions, err error) {
				assert.Error(t, err)
				assert.Equal(t, "no paths specified", err.Error())
			},
		},
		{
			name: "ref does not exist",
			mod: sp.Module{
				Instance:  "instance",
				Name:      "name",
				Namespace: "default",
				Registry:  "https://github.com/owner/repo",
				Values:    ctx.CompileString(`{paths: ["project/file.yaml"]}`),
				Version:   "foo",
			},
			files: map[string]string{
				"project/file.yaml": "file contents",
			},
			validate: func(t *testing.T, got []byte, opts remoteOptions, err error) {
				assert.Error(t, err)
				assert.Equal(t, "failed to checkout ref foo: reference not found: foo is not a valid commit hash, branch, or tag", err.Error())
			},
		},
		{
			name: "path does not exist",
			mod: sp.Module{
				Instance:  "instance",
				Name:      "name",
				Namespace: "default",
				Registry:  "https://github.com/owner/repo",
				Values:    ctx.CompileString(`{paths: ["project/file.yaml"]}`),
				Version:   "master",
			},
			files: map[string]string{
				"project/other.yaml": "file contents",
			},
			validate: func(t *testing.T, got []byte, opts remoteOptions, err error) {
				assert.Error(t, err)
				assert.Equal(t, "path project/file.yaml does not exist in git repo", err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remote, opts, err := newMockGitRemoteInterface(tt.files)
			require.NoError(t, err)

			mg := GitManifestGenerator{
				fs:     bfs.NewInMemoryFs(),
				logger: testutils.NewNoopLogger(),
				remote: remote,
			}
			got, err := mg.Generate(tt.mod, getRaw(tt.mod), "")
			tt.validate(t, got, opts, err)
		})
	}
}

type remoteOptions struct {
	Clone *gg.CloneOptions
	Push  *gg.PushOptions
}

func newMockGitRemoteInterface(
	files map[string]string,
) (remote.GitRemoteInteractor, remoteOptions, error) {
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
		}, remoteOptions{
			Clone: &cloneOpts,
			Push:  &pushOpts,
		}, nil
}

func getRaw(m sp.Module) cue.Value {
	ctx := cuecontext.New()
	return ctx.Encode(m)
}

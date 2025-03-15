package deployer

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	tu "github.com/input-output-hk/catalyst-forge/lib/project/utils/test"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	gr "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployerCreateDeployment(t *testing.T) {
	type testResult struct {
		cloneOpts *gg.CloneOptions
		deployer  Deployer
		err       error
		fs        fs.Filesystem
		result    *Deployment
	}

	gitPassword := "password"
	manifestContent := "manifest"

	tests := []struct {
		name     string
		project  string
		bundle   sp.ModuleBundle
		cfg      DeployerConfig
		files    map[string]string
		validate func(t *testing.T, r testResult)
	}{
		{
			name:    "success",
			project: "project",
			bundle: sp.ModuleBundle{
				Env: "test",
				Modules: map[string]sp.Module{
					"main": {
						Instance:  "instance",
						Name:      "module",
						Namespace: "default",
						Registry:  "registry",
						Type:      "kcl",
						Values:    map[string]string{"key": "value"},
						Version:   "v1.0.0",
					},
				},
			},
			cfg: makeConfig(),
			files: map[string]string{
				"root/test/project/env.mod.cue": `main: values: { key: "value" }`,
			},
			validate: func(t *testing.T, r testResult) {
				require.NoError(t, r.err)

				e, err := r.fs.Exists(mkPath("test", "project", "main.yaml"))
				require.NoError(t, err)
				assert.True(t, e)

				e, err = r.fs.Exists(mkPath("test", "project", "bundle.cue"))
				require.NoError(t, err)
				assert.True(t, e)

				c, err := r.fs.ReadFile(mkPath("test", "project", "main.yaml"))
				require.NoError(t, err)
				assert.Equal(t, manifestContent, string(c))

				c, err = r.fs.ReadFile(mkPath("test", "project", "bundle.cue"))
				require.NoError(t, err)
				assert.Equal(t, r.result.RawBundle, c)

				cfg := makeConfig()
				auth := r.cloneOpts.Auth.(*http.BasicAuth)
				assert.Equal(t, gitPassword, auth.Password)
				assert.Equal(t, cfg.Git.Url, r.cloneOpts.URL)
				assert.Equal(t, fmt.Sprintf("refs/heads/%s", cfg.Git.Ref), r.cloneOpts.ReferenceName.String())
			},
		},
		{
			name:    "dry run with extra files",
			project: "project",
			bundle: sp.ModuleBundle{
				Env: "test",
				Modules: map[string]sp.Module{"main": {
					Instance:  "instance",
					Name:      "module",
					Namespace: "default",
					Registry:  "registry",
					Type:      "kcl",
					Values:    map[string]string{"key": "value"},
					Version:   "v1.0.0",
				},
				},
			},
			cfg: makeConfig(),
			files: map[string]string{
				"root/test/project/extra.yaml": "extra",
			},
			validate: func(t *testing.T, r testResult) {
				require.NoError(t, r.err)

				e, err := r.fs.Exists(mkPath("test", "project", "main.yaml"))
				require.NoError(t, err)
				assert.True(t, e)

				e, err = r.fs.Exists(mkPath("test", "project", "bundle.cue"))
				require.NoError(t, err)
				assert.True(t, e)

				e, err = r.fs.Exists(mkPath("test", "project", "extra.yaml"))
				require.NoError(t, err)
				assert.False(t, e)

				rr := r.result.Repo.Raw()
				wt, err := rr.Worktree()
				require.NoError(t, err)
				st, err := wt.Status()
				require.NoError(t, err)
				fst := st.File("root/test/project/extra.yaml")
				assert.Equal(t, fst.Staging, gg.Deleted)

				fst = st.File("root/test/project/main.yaml")
				assert.Equal(t, fst.Staging, gg.Added)

				fst = st.File("root/test/project/bundle.cue")
				assert.Equal(t, fst.Staging, gg.Added)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()

			remote, opts, err := tu.NewMockGitRemoteInterface(tt.files)
			require.NoError(t, err)

			gen := tu.NewMockGenerator(manifestContent)
			ss := tu.NewMockSecretStore(map[string]string{"token": gitPassword})

			d := Deployer{
				cfg:    tt.cfg,
				ctx:    cuecontext.New(),
				fs:     fs,
				gen:    gen,
				logger: testutils.NewNoopLogger(),
				remote: remote,
				ss:     ss,
			}

			bundle := deployment.ModuleBundle{
				Bundle: tt.bundle,
				Raw:    getRaw(tt.bundle),
			}

			result, err := d.CreateDeployment(tt.project, bundle, WithFS(fs))
			tt.validate(t, testResult{
				cloneOpts: opts.Clone,
				deployer:  d,
				err:       err,
				fs:        fs,
				result:    result,
			})
		})
	}
}

func TestDeploymentCommit(t *testing.T) {
	fs := billy.NewInMemoryFs()

	remote, opts, err := tu.NewMockGitRemoteInterface(nil)
	require.NoError(t, err)

	repo, err := gr.NewGitRepo(
		"",
		testutils.NewNoopLogger(),
		gr.WithFS(fs),
		gr.WithGitRemoteInteractor(remote),
		gr.WithAuth("username", "password"),
	)
	require.NoError(t, err)

	require.NoError(t, repo.Init())
	require.NoError(t, repo.WriteFile("test.txt", []byte("test")))
	require.NoError(t, repo.StageFile("test.txt"))

	deployment := &Deployment{
		Repo:    repo,
		Project: "project",
		logger:  testutils.NewNoopLogger(),
	}

	err = deployment.Commit()
	require.NoError(t, err)

	rr := repo.Raw()
	head, err := rr.Head()
	require.NoError(t, err)

	cm, err := rr.CommitObject(head.Hash())
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf(GIT_MESSAGE, "project"), cm.Message)
	assert.Equal(t, GIT_NAME, cm.Author.Name)
	assert.Equal(t, GIT_EMAIL, cm.Author.Email)

	auth := opts.Push.Auth.(*http.BasicAuth)
	assert.Equal(t, "username", auth.Username)
	assert.Equal(t, "password", auth.Password)
}

func getRaw(bundle sp.ModuleBundle) cue.Value {
	ctx := cuecontext.New()
	return ctx.Encode(bundle)
}

func mkPath(env, project, file string) string {
	return fmt.Sprintf("/repo/root/%s/%s/%s", env, project, file)
}

func makeConfig() DeployerConfig {
	return DeployerConfig{
		Git: DeployerConfigGit{
			Creds: sc.Secret{
				Provider: "local",
				Path:     "key",
			},
			Ref: "main",
			Url: "url",
		},
		RootDir: "root",
	}
}

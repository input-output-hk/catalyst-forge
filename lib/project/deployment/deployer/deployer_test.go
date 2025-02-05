package deployer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/generator"
	dm "github.com/input-output-hk/catalyst-forge/lib/project/deployment/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	sm "github.com/input-output-hk/catalyst-forge/lib/project/secrets/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/utils"
	rm "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployerDeploy(t *testing.T) {
	newProject := func(name string, bundle schema.DeploymentModuleBundle) project.Project {
		return project.Project{
			Blueprint: schema.Blueprint{
				Project: schema.Project{
					Deployment: schema.Deployment{
						Modules: bundle,
					},
				},
				Global: schema.Global{
					Deployment: schema.GlobalDeployment{
						Environment: "test",
						Repo: schema.GlobalDeploymentRepo{
							Ref: "main",
							Url: "url",
						},
						Root: "root",
					},
					CI: schema.GlobalCI{
						Providers: schema.Providers{
							Git: schema.ProviderGit{
								Credentials: &schema.Secret{
									Provider: "local",
									Path:     "key",
								},
							},
						},
					},
				},
			},
			Name: name,
		}
	}

	type testResult struct {
		cloneOpts *gg.CloneOptions
		deployer  Deployer
		err       error
		fs        afero.Fs
		repo      *gg.Repository
	}

	tests := []struct {
		name     string
		project  project.Project
		files    map[string]string
		dryrun   bool
		validate func(t *testing.T, r testResult)
	}{
		{
			name: "success",
			project: newProject(
				"project",
				schema.DeploymentModuleBundle{
					"main": {
						Instance:  "instance",
						Name:      utils.StringPtr("module"),
						Namespace: "default",
						Registry:  utils.StringPtr("registry"),
						Values:    map[string]string{"key": "value"},
						Version:   utils.StringPtr("v1.0.0"),
					},
				},
			),
			files:  nil,
			dryrun: false,
			validate: func(t *testing.T, r testResult) {
				require.NoError(t, r.err)

				e, err := afero.Exists(r.fs, "/repo/root/test/apps/project/main.yaml")
				require.NoError(t, err)
				assert.True(t, e)

				e, err = afero.Exists(r.fs, "/repo/root/test/apps/project/mod.cue")
				require.NoError(t, err)
				assert.True(t, e)

				c, err := afero.ReadFile(r.fs, "/repo/root/test/apps/project/main.yaml")
				require.NoError(t, err)
				assert.Equal(t, "manifest", string(c))

				mod := `{
	main: {
		instance:  "instance"
		name:      "module"
		namespace: "default"
		registry:  "registry"
		values: {
			key: "value"
		}
		version: "v1.0.0"
	}
}`
				c, err = afero.ReadFile(r.fs, "/repo/root/test/apps/project/mod.cue")
				require.NoError(t, err)
				assert.Equal(t, mod, string(c))

				auth := r.cloneOpts.Auth.(*http.BasicAuth)
				assert.Equal(t, "value", auth.Password)
				assert.Equal(t, "url", r.cloneOpts.URL)
				assert.Equal(t, "refs/heads/main", r.cloneOpts.ReferenceName.String())

				head, err := r.repo.Head()
				require.NoError(t, err)

				cm, err := r.repo.CommitObject(head.Hash())
				require.NoError(t, err)
				assert.Equal(t, fmt.Sprintf(GIT_MESSAGE, "project"), cm.Message)
				assert.Equal(t, GIT_NAME, cm.Author.Name)
				assert.Equal(t, GIT_EMAIL, cm.Author.Email)
			},
		},
		{
			name: "dry run with extra files",
			project: newProject(
				"project",
				schema.DeploymentModuleBundle{
					"main": {
						Instance:  "instance",
						Name:      utils.StringPtr("module"),
						Namespace: "default",
						Registry:  utils.StringPtr("registry"),
						Values:    map[string]string{"key": "value"},
						Version:   utils.StringPtr("v1.0.0"),
					},
				},
			),
			files: map[string]string{
				"/repo/root/test/apps/project/extra.yaml": "extra",
			},
			dryrun: true,
			validate: func(t *testing.T, r testResult) {
				require.NoError(t, r.err)

				e, err := afero.Exists(r.fs, "/repo/root/test/apps/project/main.yaml")
				require.NoError(t, err)
				assert.True(t, e)

				e, err = afero.Exists(r.fs, "/repo/root/test/apps/project/mod.cue")
				require.NoError(t, err)
				assert.True(t, e)

				e, err = afero.Exists(r.fs, "/repo/root/test/apps/project/extra.yaml")
				require.NoError(t, err)
				assert.False(t, e)

				wt, err := r.repo.Worktree()
				require.NoError(t, err)
				st, err := wt.Status()
				require.NoError(t, err)
				fst := st.File("root/test/apps/project/extra.yaml")
				assert.Equal(t, fst.Staging, gg.Deleted)

				fst = st.File("root/test/apps/project/main.yaml")
				assert.Equal(t, fst.Staging, gg.Added)

				fst = st.File("root/test/apps/project/mod.cue")
				assert.Equal(t, fst.Staging, gg.Added)

				head, err := r.repo.Head()
				assert.NoError(t, err)
				cm, err := r.repo.CommitObject(head.Hash())
				require.NoError(t, err)
				assert.Equal(t, "initial commit", cm.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			var opts *gg.CloneOptions
			var repo *gg.Repository
			fs := afero.NewMemMapFs()

			remote := &rm.GitRemoteInteractorMock{
				CloneFunc: func(s storage.Storer, worktree billy.Filesystem, o *gg.CloneOptions) (*gg.Repository, error) {
					opts = o
					repo, err = gg.Init(s, worktree)
					require.NoError(t, err, "failed to init repo")

					wt, err := repo.Worktree()
					require.NoError(t, err, "failed to get worktree")

					if tt.files != nil {
						testutils.SetupFS(t, fs, tt.files)
						for path := range tt.files {
							_, err := wt.Add(strings.TrimPrefix(path, "/repo/"))
							require.NoError(t, err, "failed to add file")
						}

						_, err = wt.Commit("initial commit", &gg.CommitOptions{
							Author: &object.Signature{
								Name:  GIT_NAME,
								Email: GIT_EMAIL,
							},
						})
						require.NoError(t, err, "failed to commit")
					}

					return repo, nil
				},
				PushFunc: func(repo *gg.Repository, o *gg.PushOptions) error {
					return nil
				},
			}
			gen := generator.NewGenerator(
				&dm.ManifestGeneratorMock{
					GenerateFunc: func(mod schema.DeploymentModule) ([]byte, error) {
						return []byte("manifest"), nil
					},
				},
				testutils.NewNoopLogger(),
			)

			provider := func(logger *slog.Logger) (secrets.SecretProvider, error) {
				return &sm.SecretProviderMock{
					GetFunc: func(key string) (string, error) {
						j, err := json.Marshal(map[string]string{"token": "value"})
						require.NoError(t, err)
						return string(j), nil
					},
				}, nil
			}
			tt.project.SecretStore = secrets.NewSecretStore(
				map[secrets.Provider]func(*slog.Logger) (secrets.SecretProvider, error){
					secrets.ProviderLocal: provider,
				},
			)

			d := Deployer{
				dryrun:  tt.dryrun,
				fs:      fs,
				gen:     gen,
				logger:  testutils.NewStdoutLogger(),
				project: &tt.project,
				remote:  remote,
			}

			err = d.Deploy()
			tt.validate(t, testResult{
				cloneOpts: opts,
				deployer:  d,
				err:       err,
				fs:        fs,
				repo:      repo,
			})
		})
	}
}

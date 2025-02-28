package deployer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/generator"
	dm "github.com/input-output-hk/catalyst-forge/lib/project/deployment/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	sm "github.com/input-output-hk/catalyst-forge/lib/project/secrets/mocks"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	spr "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global/providers"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	rm "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployerDeploy(t *testing.T) {
	newProject := func(name string, bundle sp.ModuleBundle) project.Project {
		return project.Project{
			Blueprint: sb.Blueprint{
				Project: &sp.Project{
					Deployment: &sp.Deployment{
						Bundle: bundle,
					},
				},
				Global: &sg.Global{
					Deployment: &sg.Deployment{
						Repo: sg.DeploymentRepo{
							Ref: "main",
							Url: "url",
						},
						Root: "root",
					},
					Ci: &sg.CI{
						Providers: &spr.Providers{
							Git: &spr.Git{
								Credentials: sc.Secret{
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
				sp.ModuleBundle{
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
			),
			files: map[string]string{
				mkPath("dev", "project", "env.mod.cue"): `main: values: { key1: "value1" }`,
			},
			dryrun: false,
			validate: func(t *testing.T, r testResult) {
				require.NoError(t, r.err)

				e, err := afero.Exists(r.fs, mkPath("dev", "project", "main.yaml"))
				require.NoError(t, err)
				assert.True(t, e)

				e, err = afero.Exists(r.fs, mkPath("dev", "project", "mod.cue"))
				require.NoError(t, err)
				assert.True(t, e)

				c, err := afero.ReadFile(r.fs, mkPath("dev", "project", "main.yaml"))
				require.NoError(t, err)
				assert.Equal(t, "manifest", string(c))

				mod := `{
	env: "test"
	modules: {
		main: {
			instance:  "instance"
			name:      "module"
			namespace: "default"
			registry:  "registry"
			type:      "kcl"
			values: {
				key: "value"
			}
			version: "v1.0.0"
		}
	}
}`
				c, err = afero.ReadFile(r.fs, mkPath("dev", "project", "mod.cue"))
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
				sp.ModuleBundle{
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
			),
			files: map[string]string{
				mkPath("dev", "project", "extra.yaml"): "extra",
			},
			dryrun: true,
			validate: func(t *testing.T, r testResult) {
				require.NoError(t, r.err)

				e, err := afero.Exists(r.fs, mkPath("dev", "project", "main.yaml"))
				require.NoError(t, err)
				assert.True(t, e)

				e, err = afero.Exists(r.fs, mkPath("dev", "project", "mod.cue"))
				require.NoError(t, err)
				assert.True(t, e)

				e, err = afero.Exists(r.fs, mkPath("dev", "project", "extra.yaml"))
				require.NoError(t, err)
				assert.False(t, e)

				wt, err := r.repo.Worktree()
				require.NoError(t, err)
				st, err := wt.Status()
				require.NoError(t, err)
				fst := st.File("root/dev/project/extra.yaml")
				assert.Equal(t, fst.Staging, gg.Deleted)

				fst = st.File("root/dev/project/main.yaml")
				assert.Equal(t, fst.Staging, gg.Added)

				fst = st.File("root/dev/project/mod.cue")
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
				deployment.NewManifestGeneratorStore(
					map[deployment.Provider]func(*slog.Logger) deployment.ManifestGenerator{
						deployment.ProviderKCL: func(logger *slog.Logger) deployment.ManifestGenerator {
							return &dm.ManifestGeneratorMock{
								GenerateFunc: func(mod sp.Module, env string) ([]byte, error) {
									return []byte("manifest"), nil
								},
							}
						},
					},
				),
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

			tt.project.RawBlueprint = getRaw(tt.project.Blueprint)
			d := Deployer{
				ctx:     cuecontext.New(),
				dryrun:  tt.dryrun,
				fs:      fs,
				gen:     gen,
				logger:  testutils.NewNoopLogger(),
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

func getRaw(bp sb.Blueprint) blueprint.RawBlueprint {
	ctx := cuecontext.New()
	v := ctx.Encode(bp)

	return blueprint.NewRawBlueprint(v)
}

func mkPath(env, project, file string) string {
	return fmt.Sprintf("/repo/root/%s/%s/%s", env, project, file)
}

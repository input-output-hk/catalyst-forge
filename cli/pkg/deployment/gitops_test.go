package deployment

// import (
// 	"fmt"
// 	"log/slog"
// 	"testing"

// 	"cuelang.org/go/cue/cuecontext"
// 	"github.com/go-git/go-billy/v5"
// 	"github.com/go-git/go-git/v5"
// 	"github.com/go-git/go-git/v5/plumbing/transport/http"
// 	"github.com/go-git/go-git/v5/storage"
// 	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/generator"
// 	dmock "github.com/input-output-hk/catalyst-forge/lib/project/deployment/mocks"
// 	"github.com/input-output-hk/catalyst-forge/lib/project/project"
// 	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
// 	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
// 	"github.com/input-output-hk/catalyst-forge/lib/project/secrets/mocks"
// 	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// type mockGitRemote struct {
// 	cloneErr  error
// 	cloneOpts *git.CloneOptions
// 	pushErr   error
// 	pushOpts  *git.PushOptions
// 	repo      *git.Repository
// }

// func (m *mockGitRemote) Clone(s storage.Storer, worktree billy.Filesystem, o *git.CloneOptions) (*git.Repository, error) {
// 	m.cloneOpts = o
// 	if m.cloneErr != nil {
// 		return nil, m.cloneErr
// 	}

// 	return m.repo, nil
// }

// func (m *mockGitRemote) Push(repo *git.Repository, o *git.PushOptions) error {
// 	m.pushOpts = o
// 	if m.pushErr != nil {
// 		return m.pushErr
// 	}

// 	return nil
// }

// func TestDeploy(t *testing.T) {
// 	defaultParams := projectParams{
// 		projectName: "test",
// 		globalDeploy: schema.GlobalDeployment{
// 			Environment: "dev",
// 			Registries: schema.GlobalDeploymentRegistries{
// 				Modules: "registry.myserver.com",
// 			},
// 			Repo: schema.GlobalDeploymentRepo{
// 				Ref: "main",
// 				Url: "https://github.com/foo/bar",
// 			},
// 			Root: "deploy",
// 		},
// 		globalProvider: schema.ProviderGit{
// 			Credentials: &schema.Secret{
// 				Provider: "mock",
// 				Path:     "test",
// 			},
// 		},
// 		container: "mycontainer",
// 		namespace: "default",
// 		values:    `foo: "bar"`,
// 		version:   "1.0.0",
// 	}

// 	tests := []struct {
// 		name        string
// 		mock        mockGitRemote
// 		project     projectParams
// 		yaml        string
// 		execFail    bool
// 		dryrun      bool
// 		setup       func(*testing.T, *GitopsDeployer, *testutils.InMemRepo)
// 		validate    func(*testing.T, *GitopsDeployer, mockGitRemote, *testutils.InMemRepo)
// 		expectErr   bool
// 		expectedErr string
// 	}{
// 		{
// 			name:     "valid",
// 			mock:     mockGitRemote{},
// 			project:  defaultParams,
// 			yaml:     "yaml",
// 			execFail: false,
// 			dryrun:   false,
// 			setup: func(t *testing.T, deployer *GitopsDeployer, repo *testutils.InMemRepo) {
// 				deployer.token = "test"
// 				repo.MkdirAll(t, "deploy/dev/apps")
// 			},
// 			validate: func(t *testing.T, deployer *GitopsDeployer, mock mockGitRemote, repo *testutils.InMemRepo) {
// 				assert.True(t, repo.Exists(t, "deploy/dev/apps/test/main.yaml"), "main.yaml does not exist")
// 				assert.Equal(t, repo.ReadFile(t, "deploy/dev/apps/test/main.yaml"), []byte("yaml"), "main.yaml content is incorrect")

// 				head, err := repo.Repo.Head()
// 				require.NoError(t, err, "failed to get head")
// 				commit, err := repo.Repo.CommitObject(head.Hash())
// 				assert.NoError(t, err, "failed to get commit")
// 				assert.Equal(t, commit.Message, "chore: automatic deployment for test")

// 				assert.Equal(t, mock.pushOpts.Auth.(*http.BasicAuth).Password, "test")
// 			},
// 			expectErr:   false,
// 			expectedErr: "",
// 		},
// 		{
// 			name:     "dry-run",
// 			mock:     mockGitRemote{},
// 			project:  defaultParams,
// 			yaml:     "yaml",
// 			execFail: false,
// 			dryrun:   true,
// 			setup: func(t *testing.T, deployer *GitopsDeployer, repo *testutils.InMemRepo) {
// 				deployer.token = "test"
// 				repo.MkdirAll(t, "deploy/dev/apps")
// 			},
// 			validate: func(t *testing.T, deployer *GitopsDeployer, mock mockGitRemote, repo *testutils.InMemRepo) {
// 				assert.True(t, repo.Exists(t, "deploy/dev/apps/test/main.yaml"), "main.yaml does not exist")
// 				assert.Equal(t, repo.ReadFile(t, "deploy/dev/apps/test/main.yaml"), []byte("yaml"), "main.yaml content is incorrect")

// 				_, err := repo.Repo.Head()
// 				require.Error(t, err) // No commit should be made
// 			},
// 			expectErr:   false,
// 			expectedErr: "",
// 		},
// 		{
// 			name:     "no changes",
// 			mock:     mockGitRemote{},
// 			project:  defaultParams,
// 			yaml:     "yaml",
// 			execFail: false,
// 			setup: func(t *testing.T, deployer *GitopsDeployer, repo *testutils.InMemRepo) {
// 				mod := `{
// 	name:      "mycontainer"
// 	namespace: "default"
// 	values: {
// 		foo: "bar"
// 	}
// 	version: "1.0.0"
// }`
// 				repo.MkdirAll(t, "deploy/dev/apps/test")
// 				repo.AddFile(t, "deploy/dev/apps/test/main.yaml", string("yaml"))
// 				repo.AddFile(t, "deploy/dev/apps/test/main.mod.cue", mod)
// 				repo.Commit(t, "initial commit")
// 			},
// 			validate:    func(t *testing.T, deployer *GitopsDeployer, mock mockGitRemote, repo *testutils.InMemRepo) {},
// 			expectErr:   true,
// 			expectedErr: ErrNoChanges.Error(),
// 		},
// 		{
// 			name:     "extra files",
// 			mock:     mockGitRemote{},
// 			project:  defaultParams,
// 			yaml:     "yaml",
// 			execFail: false,
// 			setup: func(t *testing.T, deployer *GitopsDeployer, repo *testutils.InMemRepo) {
// 				mod := `{
// 	name:      "mycontainer"
// 	namespace: "default"
// 	values: {
// 		foo: "bar"
// 	}
// 	version: "1.0.0"
// }`
// 				repo.MkdirAll(t, "deploy/dev/apps/test")
// 				repo.AddFile(t, "deploy/dev/apps/test/main.yaml", string("yaml"))
// 				repo.AddFile(t, "deploy/dev/apps/test/main.mod.cue", mod)
// 				repo.AddFile(t, "deploy/dev/apps/test/bad.yaml", string("bad"))
// 				repo.Commit(t, "initial commit")
// 			},
// 			validate: func(t *testing.T, deployer *GitopsDeployer, mock mockGitRemote, repo *testutils.InMemRepo) {
// 				assert.False(t, repo.Exists(t, "deploy/dev/apps/test/bad.yaml"), "bad.yaml does not exist")
// 			},
// 			expectErr:   false,
// 			expectedErr: "",
// 		},
// 		{
// 			name:        "no environment folder",
// 			mock:        mockGitRemote{},
// 			project:     defaultParams,
// 			setup:       func(t *testing.T, deployer *GitopsDeployer, repo *testutils.InMemRepo) {},
// 			validate:    func(t *testing.T, deployer *GitopsDeployer, mock mockGitRemote, repo *testutils.InMemRepo) {},
// 			expectErr:   true,
// 			expectedErr: "environment path does not exist: deploy/dev/apps",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			repo := testutils.NewInMemRepo(t)
// 			gen := generator.NewGenerator(&dmock.ManifestGeneratorMock{
// 				GenerateFunc: func(mod schema.DeploymentModule, instance, registry string) ([]byte, error) {
// 					return []byte(tt.yaml), nil
// 				},
// 			}, testutils.NewNoopLogger())
// 			deployer := GitopsDeployer{
// 				dryrun:      tt.dryrun,
// 				fs:          repo.Fs,
// 				gen:         gen,
// 				logger:      testutils.NewNoopLogger(),
// 				project:     newTestProject(tt.project),
// 				remote:      &tt.mock,
// 				repo:        repo.Repo,
// 				secretStore: nil,
// 				worktree:    repo.Worktree,
// 			}

// 			tt.setup(t, &deployer, &repo)

// 			err := deployer.Deploy()
// 			if testutils.AssertError(t, err, tt.expectErr, tt.expectedErr) {
// 				return
// 			}

// 			tt.validate(t, &deployer, tt.mock, &repo)
// 		})
// 	}
// }

// func TestLoad(t *testing.T) {
// 	defaultParams := projectParams{
// 		projectName: "test",
// 		globalDeploy: schema.GlobalDeployment{
// 			Environment: "dev",
// 			Registries: schema.GlobalDeploymentRegistries{
// 				Modules: "registry.myserver.com",
// 			},
// 			Repo: schema.GlobalDeploymentRepo{
// 				Ref: "main",
// 				Url: "https://github.com/foo/bar",
// 			},
// 			Root: "deploy",
// 		},
// 		globalProvider: schema.ProviderGit{
// 			Credentials: &schema.Secret{
// 				Provider: "mock",
// 				Path:     "test",
// 			},
// 		},
// 		container: "mycontainer",
// 		namespace: "default",
// 		values:    `foo: "bar"`,
// 		version:   "1.0.0",
// 	}

// 	tests := []struct {
// 		name        string
// 		mock        mockGitRemote
// 		project     projectParams
// 		store       *secrets.SecretStore
// 		validate    func(t *testing.T, mock mockGitRemote, project *project.Project, deployer *GitopsDeployer)
// 		expectErr   bool
// 		expectedErr string
// 	}{
// 		{
// 			name:    "valid",
// 			mock:    mockGitRemote{},
// 			project: defaultParams,
// 			store:   newMockSecretStore("test", `{"token":"test"}`),
// 			validate: func(t *testing.T, mock mockGitRemote, project *project.Project, deployer *GitopsDeployer) {
// 				assert.Equal(t, deployer.token, "test")
// 				assert.Equal(t, mock.cloneOpts.URL, project.Blueprint.Global.Deployment.Repo.Url)
// 				assert.Equal(t, string(mock.cloneOpts.ReferenceName), fmt.Sprintf("refs/heads/%s", project.Blueprint.Global.Deployment.Repo.Ref))
// 				assert.Equal(t, mock.cloneOpts.Auth.(*http.BasicAuth).Password, "test")
// 			},
// 			expectErr:   false,
// 			expectedErr: "",
// 		},
// 		{
// 			name:        "clone error",
// 			mock:        mockGitRemote{cloneErr: fmt.Errorf("clone error")},
// 			project:     defaultParams,
// 			store:       newMockSecretStore("test", `{"token":"test"}`),
// 			expectErr:   true,
// 			expectedErr: "could not clone repository: clone error",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			tt.mock.repo = testutils.NewInMemRepo(t).Repo
// 			project := newTestProject(tt.project)
// 			deployer := GitopsDeployer{
// 				logger:      testutils.NewNoopLogger(),
// 				project:     project,
// 				remote:      &tt.mock,
// 				secretStore: newMockSecretStore("test", `{"token":"test"}`),
// 			}

// 			err := deployer.Load()
// 			if testutils.AssertError(t, err, tt.expectErr, tt.expectedErr) {
// 				return
// 			}

// 			tt.validate(t, tt.mock, project, &deployer)
// 		})
// 	}
// }

// type projectParams struct {
// 	projectName    string
// 	globalDeploy   schema.GlobalDeployment
// 	globalProvider schema.ProviderGit
// 	container      string
// 	namespace      string
// 	values         string
// 	version        string
// }

// func newTestProject(p projectParams) *project.Project {
// 	ctx := cuecontext.New()
// 	return &project.Project{
// 		Name: p.projectName,
// 		Blueprint: schema.Blueprint{
// 			Global: schema.Global{
// 				Deployment: p.globalDeploy,
// 				CI: schema.GlobalCI{
// 					Providers: schema.Providers{
// 						Git: p.globalProvider,
// 					},
// 				},
// 			},
// 			Project: schema.Project{
// 				Deployment: schema.Deployment{
// 					Modules: map[string]schema.DeploymentModule{
// 						"main": {
// 							Name:      p.container,
// 							Namespace: p.namespace,
// 							Values:    ctx.CompileString(p.values),
// 							Version:   p.version,
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}
// }

// func newMockSecretStore(secretPath string, value string) *secrets.SecretStore {
// 	provider := &mocks.SecretProviderMock{
// 		GetFunc: func(path string) (string, error) {
// 			if path == secretPath {
// 				return value, nil
// 			} else {
// 				return "", fmt.Errorf("secret not found")
// 			}
// 		},
// 	}
// 	store := secrets.NewSecretStore(map[secrets.Provider]func(*slog.Logger) (secrets.SecretProvider, error){
// 		secrets.Provider("mock"): func(logger *slog.Logger) (secrets.SecretProvider, error) {
// 			return provider, nil
// 		},
// 	})

// 	return &store
// }

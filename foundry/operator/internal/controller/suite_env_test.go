package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"cuelang.org/go/cue/cuecontext"
	"github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/deployments"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/deployments/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/github"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/releases"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
	"github.com/input-output-hk/catalyst-forge/foundry/operator/pkg/config"
	"github.com/input-output-hk/catalyst-forge/foundry/operator/pkg/handlers"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/deployer"
	tu "github.com/input-output-hk/catalyst-forge/lib/project/utils/test"
	"github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	bfs "github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
	rm "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type mockEnv struct {
	blueprint             *sb.Blueprint
	blueprintRaw          string
	bundleRaw             string
	config                config.OperatorConfig
	deployFs              *bfs.BillyFs
	manifestContent       string
	mockClient            client.Client
	mockDeploymentsClient *mocks.DeploymentsClientInterfaceMock
	mockEventsClient      *mocks.EventsClientInterfaceMock
	mockClock             *MockClock
	mockDeploymentHandler *handlers.ReleaseDeploymentHandler
	mockManifestStore     deployment.ManifestGeneratorStore
	mockRemote            *rm.GitRemoteInteractorMock
	mockRepoHandler       *handlers.RepoHandler
	mockSecretStore       secrets.SecretStore
	releaseDeploymentObj  *foundryv1alpha1.ReleaseDeployment
	releaseDeployment     *deployments.ReleaseDeployment
	sourceFs              *bfs.BillyFs
	sourceRepo            *gg.Repository
}

func (m *mockEnv) Init(sourceFiles, deployFiles map[string]string, k8sClient k8sclient.Client) {
	var err error

	// Setup filesystems
	m.deployFs = bfs.NewInMemoryFs()
	m.sourceFs = bfs.NewInMemoryFs()

	// Setup the default mock values
	m.blueprint = newBlueprint()
	m.blueprintRaw = newRawBlueprint()
	m.bundleRaw = newRawBundle()
	m.config = newConfig()
	m.manifestContent = "manifest"

	m.releaseDeploymentObj = &foundryv1alpha1.ReleaseDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-001-123456789",
			Namespace: "default",
		},
		Spec: foundryv1alpha1.ReleaseDeploymentSpec{
			ID:        "project-001-123456789",
			ReleaseID: "project-001",
		},
	}
	m.releaseDeployment = &deployments.ReleaseDeployment{
		ID:     "project-001-123456789",
		Status: deployments.DeploymentStatusPending,
		Release: &releases.Release{
			ID:          "project-001",
			SourceRepo:  "github.com/repo/source",
			Project:     "project",
			ProjectPath: "project",
			Bundle:      "bundle",
		},
	}

	// Setup the mock category clients
	m.mockDeploymentsClient = &mocks.DeploymentsClientInterfaceMock{}
	m.mockEventsClient = &mocks.EventsClientInterfaceMock{}

	// Setup the mock deployments client
	m.mockDeploymentsClient.GetFunc = func(ctx context.Context, releaseID string, deployID string) (*deployments.ReleaseDeployment, error) {
		return m.releaseDeployment, nil
	}
	m.mockDeploymentsClient.IncrementAttemptsFunc = func(ctx context.Context, releaseID string, deployID string) (*deployments.ReleaseDeployment, error) {
		m.releaseDeployment.Attempts++
		return m.releaseDeployment, nil
	}
	m.mockDeploymentsClient.UpdateFunc = func(ctx context.Context, releaseID string, deployment *deployments.ReleaseDeployment) (*deployments.ReleaseDeployment, error) {
		m.releaseDeployment = deployment
		return m.releaseDeployment, nil
	}

	// Setup the mock events client
	m.mockEventsClient.AddFunc = func(ctx context.Context, releaseID string, deployID string, name string, message string) (*deployments.ReleaseDeployment, error) {
		m.releaseDeployment.Events = append(m.releaseDeployment.Events, deployments.DeploymentEvent{
			Name:    name,
			Message: message,
		})
		return m.releaseDeployment, nil
	}

	// Create a mock client that returns our category clients
	m.mockClient = &mockClient{
		deployments: m.mockDeploymentsClient,
		events:      m.mockEventsClient,
	}

	// Setup the mock Git remote
	m.mockRemote, err = newMockGitRemoteInterface(map[string]map[string]string{
		m.releaseDeployment.Release.SourceRepo: sourceFiles,
		m.blueprint.Global.Deployment.Repo.Url: deployFiles,
	})
	Expect(err).ToNot(HaveOccurred())

	// Setup the mock manifest store
	m.mockManifestStore = tu.NewMockManifestStore(m.manifestContent)

	// Setup the mock secret store
	m.mockSecretStore = tu.NewMockSecretStore(map[string]string{"token": "value"})

	// Setup the mock handlers
	m.mockClock = NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	m.mockDeploymentHandler = handlers.NewReleaseDeploymentHandlerWithClock(context.Background(), m.mockClient, k8sClient, m.mockClock)
	m.mockRepoHandler = handlers.NewRepoHandler(m.deployFs, m.sourceFs, testutils.NewNoopLogger(), m.mockRemote, "token")

	// Initialize the source repository (to properly set the source commit)
	m.sourceRepo, err = initRepo(
		*m.sourceFs,
		makeCachePath(m.releaseDeployment.Release.SourceRepo),
		m.releaseDeployment.Release.SourceRepo,
		m.mockRemote,
	)
	Expect(err).ToNot(HaveOccurred())
	head, err := m.sourceRepo.Head()
	Expect(err).ToNot(HaveOccurred())
	m.releaseDeployment.Release.SourceCommit = head.Hash().String()
}

// mockClient implements client.Client for testing
type mockClient struct {
	deployments deployments.DeploymentsClientInterface
	events      deployments.EventsClientInterface
}

func (m *mockClient) Users() users.UsersClientInterface {
	return nil
}

func (m *mockClient) Roles() users.RolesClientInterface {
	return nil
}

func (m *mockClient) Keys() users.KeysClientInterface {
	return nil
}

func (m *mockClient) Auth() auth.AuthClientInterface {
	return nil
}

func (m *mockClient) Github() github.GithubClientInterface {
	return nil
}

func (m *mockClient) Releases() releases.ReleasesClientInterface {
	return nil
}

func (m *mockClient) Aliases() releases.AliasesClientInterface {
	return nil
}

func (m *mockClient) Deployments() deployments.DeploymentsClientInterface {
	return m.deployments
}

func (m *mockClient) Events() deployments.EventsClientInterface {
	return m.events
}

func (m *mockEnv) ConfigureController(ctrl *ReleaseDeploymentReconciler) {
	ctrl.Config = m.config
	ctrl.DeploymentHandler = m.mockDeploymentHandler
	ctrl.ManifestStore = m.mockManifestStore
	ctrl.Remote = m.mockRemote
	ctrl.RepoHandler = m.mockRepoHandler
	ctrl.SecretStore = m.mockSecretStore
}

func initRepo(fs bfs.BillyFs, path, url string, remote remote.GitRemoteInteractor) (*gg.Repository, error) {
	mock, ok := remote.(*rm.GitRemoteInteractorMock)
	if !ok {
		return nil, fmt.Errorf("failed to cast remote to mock")
	}

	gfs, err := fs.Raw().Chroot(filepath.Join(path, ".git"))
	if err != nil {
		return nil, fmt.Errorf("failed to chroot fs: %w", err)
	}

	wfs, err := fs.Raw().Chroot(path)
	if err != nil {
		return nil, fmt.Errorf("failed to chroot fs: %w", err)
	}

	storage := filesystem.NewStorage(gfs, cache.NewObjectLRUDefault())
	return mock.CloneFunc(storage, wfs, &gg.CloneOptions{
		URL: url,
	})
}

func newMockGitRemoteInterface(
	repos map[string]map[string]string,
) (*rm.GitRemoteInteractorMock, error) {
	return &rm.GitRemoteInteractorMock{
		CloneFunc: func(s storage.Storer, worktree billy.Filesystem, o *gg.CloneOptions) (*gg.Repository, error) {
			repo, err := gg.Init(s, worktree)
			if err != nil {
				return nil, fmt.Errorf("failed to init repo: %w", err)
			}

			wt, err := repo.Worktree()
			if err != nil {
				return nil, fmt.Errorf("failed to get worktree: %w", err)
			}

			files, ok := repos[o.URL]
			if ok {
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
		FetchFunc: func(repo *gg.Repository, o *gg.FetchOptions) error {
			return nil
		},
		PullFunc: func(repo *gg.Repository, o *gg.PullOptions) error {
			return nil
		},
		PushFunc: func(repo *gg.Repository, o *gg.PushOptions) error {
			return nil
		},
	}, nil
}

func newBlueprint() *sb.Blueprint {
	var blueprint sb.Blueprint
	cc := cuecontext.New()
	v := cc.CompileString(newRawBlueprint())
	Expect(v.Decode(&blueprint)).NotTo(HaveOccurred())

	return &blueprint
}

func newConfig() config.OperatorConfig {
	bp := newBlueprint()
	return config.OperatorConfig{
		Deployer: deployer.DeployerConfig{
			Git: deployer.DeployerConfigGit{
				Creds: sc.Secret{
					Provider: "local",
					Path:     "key",
				},
				Ref: bp.Global.Deployment.Repo.Ref,
				Url: bp.Global.Deployment.Repo.Url,
			},
			RootDir: bp.Global.Deployment.Root,
		},
		MaxAttempts: 3,
	}
}

func newRawBlueprint() string {
	return `
		{
			project: {
				name: "project"
				deployment: {
					on: {}
					bundle: {
						env: "test"
						modules: {
							main: {
								name: "module"
								version: "v1.0.0"
								values: {
									foo: "bar"
								}
							}
						}
					}
				}
			}
			global: {
				deployment: {
					registries: {
						containers: "registry.com"
						modules: "registry.com"
					}
					repo: {
						ref: "master"
						url: "github.com/org/repo"
					}
					root: "root"
				}
			}
		}
	`
}

func newRawBundle() string {
	return `{
	_#def
	_#def: {
		env: string
		modules: {
			[string]: {
				instance?: string
				name?:     string
				namespace: string | *"default"
				path?:     string
				registry?: string
				type:      string | *"kcl"
				values?:   _
				version?:  string
			}
		}
	} & {
		env: "test"
		modules: {
			main: {
				name:     "module"
				version:  "v1.0.0"
				instance: "project"
				registry: "registry.com"
				values: {
					foo: "bar"
				}
			}
		}
	}
}`
}

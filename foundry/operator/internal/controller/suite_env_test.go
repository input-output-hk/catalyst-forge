package controller

import (
	"context"
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue/cuecontext"
	"github.com/go-git/go-billy/v5"
	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	api "github.com/input-output-hk/catalyst-forge/foundry/api/client"
	am "github.com/input-output-hk/catalyst-forge/foundry/api/client/mocks"
	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
	"github.com/input-output-hk/catalyst-forge/foundry/operator/pkg/config"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/deployer"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	tu "github.com/input-output-hk/catalyst-forge/lib/project/utils/test"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	bfs "github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
	rm "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote/mocks"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockEnv struct {
	blueprint            *sb.Blueprint
	blueprintRaw         string
	bundleRaw            string
	config               config.OperatorConfig
	deployFs             *bfs.BillyFs
	manifestContent      string
	mockClient           *am.ClientMock
	mockManifestStore    deployment.ManifestGeneratorStore
	mockRemote           *rm.GitRemoteInteractorMock
	mockSecretStore      secrets.SecretStore
	releaseDeploymentObj *foundryv1alpha1.ReleaseDeployment
	releaseDeployment    *api.ReleaseDeployment
	sourceFs             *bfs.BillyFs
	sourceRepo           *gg.Repository
}

func (m *mockEnv) Init() {
	var err error

	// Setup filesystems
	m.deployFs = bfs.NewInMemoryFs()
	m.sourceFs = bfs.NewInMemoryFs()

	// Setup the default mock values
	m.blueprint = NewBlueprint()
	m.blueprintRaw = NewRawBlueprint()
	m.bundleRaw = NewRawBundle()
	m.config = NewConfig()
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
	m.releaseDeployment = &api.ReleaseDeployment{
		ID:     "project-001-123456789",
		Status: api.DeploymentStatusPending,
		Release: &api.Release{
			ID:          "project-001",
			SourceRepo:  "github.com/repo/source",
			Project:     "project",
			ProjectPath: "project",
			Bundle:      "bundle",
		},
	}

	// Setup the mock API client
	m.mockClient = &am.ClientMock{
		GetDeploymentFunc: func(ctx context.Context, releaseID string, deployID string) (*api.ReleaseDeployment, error) {
			return m.releaseDeployment, nil
		},
		UpdateDeploymentFunc: func(ctx context.Context, releaseID string, deployment *api.ReleaseDeployment) (*api.ReleaseDeployment, error) {
			m.releaseDeployment = deployment
			return m.releaseDeployment, nil
		},
	}

	// Setup the mock Git remote
	m.mockRemote, err = NewMockGitRemoteInterface(map[string]map[string]string{
		m.releaseDeployment.Release.SourceRepo: {
			"project/blueprint.cue": m.blueprintRaw,
		},
		m.blueprint.Global.Deployment.Repo.Url: {
			"root/test/project/env.cue": `main: values: { key1: "value1" }`,
		},
	})
	Expect(err).ToNot(HaveOccurred())

	// Setup the mock manifest store
	m.mockManifestStore = tu.NewMockManifestStore(m.manifestContent)

	// Setup the mock secret store
	m.mockSecretStore = tu.NewMockSecretStore(map[string]string{"token": "value"})

	// Initialize the source repository (to properly set the source commit)
	m.sourceRepo, err = InitRepo(
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

func (m *mockEnv) ConfigureController(ctrl *ReleaseDeploymentReconciler) {
	ctrl.ApiClient = m.mockClient
	ctrl.Config = m.config
	ctrl.FsDeploy = m.deployFs
	ctrl.FsSource = m.sourceFs
	ctrl.ManifestStore = m.mockManifestStore
	ctrl.Remote = m.mockRemote
	ctrl.SecretStore = m.mockSecretStore
}

func InitRepo(fs bfs.BillyFs, path, url string, remote remote.GitRemoteInteractor) (*gg.Repository, error) {
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

func NewMockGitRemoteInterface(
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

func NewBlueprint() *sb.Blueprint {
	var blueprint sb.Blueprint
	cc := cuecontext.New()
	v := cc.CompileString(NewRawBlueprint())
	Expect(v.Decode(&blueprint)).NotTo(HaveOccurred())

	return &blueprint
}

func NewConfig() config.OperatorConfig {
	bp := NewBlueprint()
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
	}
}

func NewRawBlueprint() string {
	return `
		{
			version: "1.0"
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

func NewRawBundle() string {
	return `{
	_#def
	_#def: {
		env: string | *"dev"
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

/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	gg "github.com/go-git/go-git/v5"
	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
	"github.com/input-output-hk/catalyst-forge/foundry/operator/pkg/config"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/deployer"
	tu "github.com/input-output-hk/catalyst-forge/lib/project/utils/test"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	bfs "github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
	rm "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	ctrl "sigs.k8s.io/controller-runtime"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx        context.Context
	cancel     context.CancelFunc
	testEnv    *envtest.Environment
	cfg        *rest.Config
	k8sClient  client.Client
	controller *ReleaseDeploymentReconciler

	gitCreds        = map[string]string{"token": "value"}
	manifestContent = "manifest"
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	err = foundryv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// setup controller
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	controller = NewMockController(k8sManager.GetClient(), k8sManager.GetScheme())
	err = controller.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}

func getNamespacedName(obj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}

func NewMockController(client client.Client, scheme *runtime.Scheme) *ReleaseDeploymentReconciler {
	// remote, err := NewMockGitRemoteInterface(map[string]map[string]string{
	// 	// Mocks the source repository where the deployment blueprint is stored
	// 	constants.release.Spec.Git.URL: {
	// 		"project/blueprint.cue": NewBlueprint(),
	// 	},

	// 	// Mocks the deployment repository where the deployment is stored
	// 	// Uses an env.cue file which should be combined with the bundle
	// 	blueprint.Global.Deployment.Repo.Url: {
	// 		"root/test/project/env.cue": `main: values: { key1: "value1" }`,
	// 	},
	// })
	// Expect(err).ToNot(HaveOccurred())

	return &ReleaseDeploymentReconciler{
		Client:        client,
		FsDeploy:      bfs.NewInMemoryFs(),
		FsSource:      bfs.NewInMemoryFs(),
		Logger:        testutils.NewNoopLogger(),
		ManifestStore: tu.NewMockManifestStore(manifestContent),
		Scheme:        scheme,
		SecretStore:   tu.NewMockSecretStore(gitCreds),
	}
}

func NewMockGitRemoteInterface(
	repos map[string]map[string]string,
) (remote.GitRemoteInteractor, error) {
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

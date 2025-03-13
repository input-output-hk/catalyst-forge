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
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	gg "github.com/go-git/go-git/v5"
	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/deployer"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/generator"
	dm "github.com/input-output-hk/catalyst-forge/lib/project/deployment/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	sm "github.com/input-output-hk/catalyst-forge/lib/project/secrets/mocks"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	rm "github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	ctrl "sigs.k8s.io/controller-runtime"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
	md        mockDeployer
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
	md = newMockDeployer()
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&ReleaseReconciler{
		Client:   k8sManager.GetClient(),
		Deployer: md.deployer,
		Scheme:   k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
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

type mockDeployer struct {
	ctx      *cue.Context
	deployer deployer.Deployer
	fs       afero.Fs
}

func newMockDeployer() mockDeployer {
	var err error
	var repo *gg.Repository

	ctx := cuecontext.New()
	fs := afero.NewMemMapFs()

	files := map[string]string{
		mkPath("test", "project", "env.mod.cue"): `main: values: { key1: "value1" }`,
	}

	remote := &rm.GitRemoteInteractorMock{
		CloneFunc: func(s storage.Storer, worktree billy.Filesystem, o *gg.CloneOptions) (*gg.Repository, error) {
			repo, err = gg.Init(s, worktree)
			Expect(err).ToNot(HaveOccurred())

			wt, err := repo.Worktree()
			Expect(err).ToNot(HaveOccurred())

			for path, content := range files {
				dir := filepath.Dir(path)

				err := fs.MkdirAll(dir, 0755)
				Expect(err).ToNot(HaveOccurred())

				err = afero.WriteFile(fs, path, []byte(content), 0644)
				Expect(err).ToNot(HaveOccurred())

				_, err = wt.Add(strings.TrimPrefix(path, "/repo/"))
				Expect(err).ToNot(HaveOccurred())
			}

			_, err = wt.Commit("initial commit", &gg.CommitOptions{
				Author: &object.Signature{
					Name:  "test",
					Email: "test@test.com",
				},
			})
			Expect(err).ToNot(HaveOccurred())

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
				Expect(err).ToNot(HaveOccurred())
				return string(j), nil
			},
		}, nil
	}

	ss := secrets.NewSecretStore(
		map[secrets.Provider]func(*slog.Logger) (secrets.SecretProvider, error){
			secrets.ProviderLocal: provider,
		},
	)

	cfg := deployer.DeployerConfig{
		Git: deployer.DeployerConfigGit{
			Creds: sc.Secret{
				Provider: "local",
				Path:     "key",
			},
			Ref: "main",
			Url: "url",
		},
		RootDir: "root",
	}

	d := deployer.NewCustomDeployer(
		cfg,
		ctx,
		fs,
		gen,
		testutils.NewNoopLogger(),
		remote,
		ss,
	)

	return mockDeployer{
		ctx:      ctx,
		deployer: d,
		fs:       fs,
	}
}

func mkPath(env, project, file string) string {
	return fmt.Sprintf("/repo/root/%s/%s/%s", env, project, file)
}

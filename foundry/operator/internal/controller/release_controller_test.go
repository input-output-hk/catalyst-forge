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
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/input-output-hk/catalyst-forge/foundry/api/client"
	am "github.com/input-output-hk/catalyst-forge/foundry/api/client/mocks"
	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
)

var _ = Describe("ReleaseDeployment Controller", func() {
	Context("When reconciling a release deployment", func() {
		const (
			timeout  = time.Second * 10
			interval = time.Millisecond * 250
		)

		var (
			ctx          = context.Background()
			blueprint    = NewBlueprint()
			blueprintRaw = NewRawBlueprint()
			bundleRaw    = NewRawBundle()
			config       = NewConfig()
		)

		Context("when the release deployment is valid", Ordered, func() {
			var (
				gotReleaseID string
				gotDeployID  string

				releaseDeploymentObj *foundryv1alpha1.ReleaseDeployment
				releaseDeployment    *api.ReleaseDeployment
			)

			BeforeAll(func() {
				releaseDeploymentObj = &foundryv1alpha1.ReleaseDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "project-001-123456789",
						Namespace: "default",
					},
					Spec: foundryv1alpha1.ReleaseDeploymentSpec{
						ID:        "project-001-123456789",
						ReleaseID: "project-001",
					},
				}
				releaseDeployment = &api.ReleaseDeployment{
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

				apiClient := am.ClientMock{
					GetDeploymentFunc: func(ctx context.Context, releaseID string, deployID string) (*api.ReleaseDeployment, error) {
						gotReleaseID = releaseID
						gotDeployID = deployID
						return releaseDeployment, nil
					},
					UpdateDeploymentStatusFunc: func(ctx context.Context, releaseID string, deployID string, status api.DeploymentStatus, reason string) error {
						releaseDeployment.Status = status
						return nil
					},
				}
				remote, err := NewMockGitRemoteInterface(map[string]map[string]string{
					releaseDeployment.Release.SourceRepo: {
						"project/blueprint.cue": blueprintRaw,
					},
					blueprint.Global.Deployment.Repo.Url: {
						"root/test/project/env.cue": `main: values: { key1: "value1" }`,
					},
				})
				Expect(err).ToNot(HaveOccurred())

				rp, err := InitRepo(
					*controller.FsSource,
					makeCachePath(releaseDeployment.Release.SourceRepo),
					releaseDeployment.Release.SourceRepo,
					remote,
				)
				Expect(err).ToNot(HaveOccurred())
				head, err := rp.Head()
				Expect(err).ToNot(HaveOccurred())
				releaseDeployment.Release.SourceCommit = head.Hash().String()

				controller.ApiClient = &apiClient
				controller.Config = config
				controller.Remote = remote

				release := &foundryv1alpha1.ReleaseDeployment{}
				err = k8sClient.Get(ctx, getNamespacedName(releaseDeploymentObj), release)
				if err != nil && errors.IsNotFound(err) {
					Expect(k8sClient.Create(ctx, releaseDeploymentObj)).To(Succeed())
				}
			})

			// AfterEach(func() {
			// 	// TODO(user): Cleanup logic after each test, like removing the resource instance.
			// 	resource := &foundryv1alpha1.Release{}
			// 	err := k8sClient.Get(ctx, typeNamespacedName, resource)
			// 	Expect(err).NotTo(HaveOccurred())

			// 	By("Cleanup the specific resource instance Release")
			// 	Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			// })

			It("should set the status to succeeded", func() {
				//release := &foundryv1alpha1.ReleaseDeployment{}
				Eventually(func(g Gomega) {
					//g.Expect(k8sClient.Get(ctx, getNamespacedName(releaseDeploymentObj), release)).To(Succeed())
					//g.Expect(release.Status.State).To(Equal("Deployed"))
					g.Expect(releaseDeployment.Status).To(Equal(api.DeploymentStatusSucceeded))
				}, timeout, interval).Should(Succeed())
			})

			It("should have called the API client to get the deployment", func() {
				Expect(gotDeployID).To(Equal(releaseDeployment.ID))
				Expect(gotReleaseID).To(Equal(releaseDeployment.Release.ID))
			})

			It("should have cloned the source repository", func() {
				path := makeCachePath(releaseDeployment.Release.SourceRepo)
				Expect(controller.FsSource.Exists(path)).To(BeTrue())
			})

			It("should have cloned the deploy repository", func() {
				path := makeCachePath(config.Deployer.Git.Url)
				Expect(controller.FsDeploy.Exists(path)).To(BeTrue())
			})

			It("should have created the deployment files", func() {
				path := makeCachePath(config.Deployer.Git.Url)
				path = filepath.Join(
					path,
					config.Deployer.RootDir,
					blueprint.Project.Deployment.Bundle.Env,
					releaseDeployment.Release.ProjectPath,
				)
				Expect(controller.FsDeploy.Exists(filepath.Join(path, "main.yaml"))).To(BeTrue())
				Expect(controller.FsDeploy.Exists(filepath.Join(path, "module.cue"))).To(BeTrue())

				got, err := controller.FsDeploy.ReadFile(filepath.Join(path, "module.cue"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(got)).To(Equal(string(bundleRaw)))

				got, err = controller.FsDeploy.ReadFile(filepath.Join(path, "main.yaml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(got)).To(Equal(string(manifestContent)))
			})
		})
	})
})

func makeCachePath(url string) string {
	pathParts := []string{xdg.CacheHome, "forge"}
	pathParts = append(pathParts, strings.Split(url, "/")...)
	return filepath.Join(pathParts...)
}

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
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"

	api "github.com/input-output-hk/catalyst-forge/foundry/api/client"
	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
)

var _ = Describe("ReleaseDeployment Controller", func() {
	Context("When reconciling a release deployment", func() {
		const (
			timeout  = time.Second * 10
			interval = time.Millisecond * 250
		)

		Context("when the release deployment is valid", Ordered, func() {
			var (
				env mockEnv
			)

			BeforeAll(func() {
				// Initialize the test environment
				env.Init(
					map[string]string{
						"project/blueprint.cue": newRawBlueprint(),
					},
					map[string]string{
						"root/test/project/env.cue": `main: values: { key1: "value1" }`,
					},
				)
				env.ConfigureController(controller)

				release := &foundryv1alpha1.ReleaseDeployment{}
				err := k8sClient.Get(ctx, getNamespacedName(env.releaseDeploymentObj), release)
				if err != nil && errors.IsNotFound(err) {
					Expect(k8sClient.Create(ctx, env.releaseDeploymentObj)).To(Succeed())
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
					g.Expect(env.releaseDeployment.Status).To(Equal(api.DeploymentStatusSucceeded))
					g.Expect(hasEvent(env.releaseDeployment.Events, "DeploymentSucceeded", "Deployment has succeeded")).To(BeTrue())
				}, timeout, interval).Should(Succeed())
			})

			It("should have called the API client to get the deployment", func() {
				Expect(len(env.mockClient.GetDeploymentCalls())).To(Equal(1))
				Expect(env.mockClient.GetDeploymentCalls()[0].DeployID).To(Equal(env.releaseDeployment.ID))
				Expect(env.mockClient.GetDeploymentCalls()[0].ReleaseID).To(Equal(env.releaseDeployment.Release.ID))
			})

			It("should have added a start event", func() {
				Expect(hasEvent(env.releaseDeployment.Events, "DeploymentStarted", "Deployment has started")).To(BeTrue())
			})

			It("should have incremented the deployment attempts", func() {
				Expect(env.releaseDeployment.Attempts).To(Equal(1))
			})

			It("should have cloned the source repository", func() {
				path := makeCachePath(env.releaseDeployment.Release.SourceRepo)
				Expect(env.sourceFs.Exists(path)).To(BeTrue())
			})

			It("should have cloned the deploy repository", func() {
				path := makeCachePath(env.config.Deployer.Git.Url)
				Expect(env.deployFs.Exists(path)).To(BeTrue())
			})

			It("should have created the deployment files", func() {
				path := makeCachePath(env.config.Deployer.Git.Url)
				path = filepath.Join(
					path,
					env.config.Deployer.RootDir,
					env.blueprint.Project.Deployment.Bundle.Env,
					env.releaseDeployment.Release.ProjectPath,
				)
				Expect(env.deployFs.Exists(filepath.Join(path, "main.yaml"))).To(BeTrue())
				Expect(env.deployFs.Exists(filepath.Join(path, "module.cue"))).To(BeTrue())

				got, err := env.deployFs.ReadFile(filepath.Join(path, "module.cue"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(got)).To(Equal(string(env.bundleRaw)))

				got, err = env.deployFs.ReadFile(filepath.Join(path, "main.yaml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(got)).To(Equal(string(env.manifestContent)))
			})
		})
	})
})

func hasEvent(events []api.DeploymentEvent, name string, message string) bool {
	for _, event := range events {
		if event.Name == name && event.Message == message {
			return true
		}
	}
	return false
}

func makeCachePath(url string) string {
	pathParts := []string{xdg.CacheHome, "forge"}
	pathParts = append(pathParts, strings.Split(url, "/")...)
	return filepath.Join(pathParts...)
}

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
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"

	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
)

var _ = Describe("Release Controller", func() {
	Context("When reconciling a release", func() {
		const (
			timeout  = time.Second * 10
			interval = time.Millisecond * 250
		)

		var (
			ctx = context.Background()
		)

		BeforeEach(func() {
			By("creating the custom resource for the Kind Release")
			release := &foundryv1alpha1.Release{}
			err := k8sClient.Get(ctx, constants.releaseNamespacedName, release)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, &constants.release)).To(Succeed())
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

		It("should successfully reconcile the resource", func() {
			release := &foundryv1alpha1.Release{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, constants.releaseNamespacedName, release)).To(Succeed())
				g.Expect(release.Status.State).To(Equal("Deployed"))
			}, timeout, interval).Should(Succeed())
		})

		It("should have cloned the source repository", func() {
			pathParts := []string{xdg.CacheHome, "forge"}
			pathParts = append(pathParts, strings.Split(constants.gitSrc.url, "/")...)
			path := filepath.Join(pathParts...)
			Expect(controller.FsSource.Exists(path)).To(BeTrue())
		})

		It("should have cloned the deploy repository", func() {
			path := fmt.Sprintf(
				"/repo/%s/%s/%s",
				constants.config.Deployer.RootDir,
				blueprint.Project.Deployment.Bundle.Env,
				constants.projectName,
			)
			// controller.FsDeploy.Walk("/", func(p string, info fs.FileInfo, err error) error {
			// 	fmt.Println(p)
			// 	return nil
			// })
			Expect(controller.FsDeploy.Exists(path)).To(BeTrue())
		})
	})
})

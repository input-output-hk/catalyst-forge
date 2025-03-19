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
	"log/slog"

	"cuelang.org/go/cue/cuecontext"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
	"github.com/input-output-hk/catalyst-forge/foundry/operator/pkg/config"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/deployer"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
)

// ReleaseReconciler reconciles a Release object
type ReleaseReconciler struct {
	client.Client
	Config        config.OperatorConfig
	FsDeploy      *billy.BillyFs
	FsSource      *billy.BillyFs
	Logger        *slog.Logger
	ManifestStore deployment.ManifestGeneratorStore
	Remote        remote.GitRemoteInteractor
	Scheme        *runtime.Scheme
	SecretStore   secrets.SecretStore
}

// +kubebuilder:rbac:groups=foundry.projectcatalyst.io,resources=releases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=foundry.projectcatalyst.io,resources=releases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=foundry.projectcatalyst.io,resources=releases/finalizers,verbs=update

func (r *ReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Load the filesystems to use for the deploy and source repositories.
	dfs, sfs := r.getFs()

	var release v1alpha1.Release
	if err := r.Get(ctx, req.NamespacedName, &release); err != nil {
		log.Error(err, "unable to fetch Release")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if release.Status.State == "Deployed" {
		log.Info("Release already deployed")
		return ctrl.Result{}, nil
	}

	release.Status.State = "Deploying"
	if err := r.Status().Update(ctx, &release); err != nil {
		log.Error(err, "unable to update Release status")
		return ctrl.Result{}, err
	}

	rp, err := repo.NewCachedRepo(
		release.Spec.Git.URL,
		r.Logger,
		repo.WithFS(sfs),
		repo.WithGitRemoteInteractor(r.Remote),
	)
	if err != nil {
		log.Error(err, "unable to create repo")
		return ctrl.Result{}, err
	}

	bundle, err := deployment.FetchBundle(rp, release.Spec.ProjectPath, r.SecretStore, r.Logger)
	if err != nil {
		log.Error(err, "unable to fetch bundle")
		return ctrl.Result{}, err
	}

	d := deployer.NewDeployer(
		r.Config.Deployer,
		r.ManifestStore,
		r.SecretStore,
		r.Logger,
		cuecontext.New(),
		deployer.WithGitRemoteInteractor(r.Remote),
	)
	deployment, err := d.CreateDeployment(release.Spec.Project, bundle, deployer.WithFS(dfs))
	if err != nil {
		log.Error(err, "unable to create deployment")
		return ctrl.Result{}, err
	}

	if err := deployment.Commit(); err != nil {
		log.Error(err, "unable to commit deployment")
		return ctrl.Result{}, err
	}

	release.Status.State = "Deployed"
	if err := r.Status().Update(ctx, &release); err != nil {
		log.Error(err, "unable to update Release status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// getFs returns the filesystems to use for the deploy and source repositories.
func (r *ReleaseReconciler) getFs() (*billy.BillyFs, *billy.BillyFs) {
	var dfs *billy.BillyFs
	if r.FsDeploy == nil {
		dfs = billy.NewInMemoryFs()
	} else {
		dfs = r.FsDeploy
	}

	var sfs *billy.BillyFs
	if r.FsSource == nil {
		sfs = billy.NewBaseOsFS()
	} else {
		sfs = r.FsSource
	}

	return dfs, sfs
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&foundryv1alpha1.Release{}).
		Named("release").
		Complete(r)
}

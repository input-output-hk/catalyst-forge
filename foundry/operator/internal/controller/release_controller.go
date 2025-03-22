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
	"log/slog"

	"cuelang.org/go/cue/cuecontext"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	api "github.com/input-output-hk/catalyst-forge/foundry/api/client"

	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
	"github.com/input-output-hk/catalyst-forge/foundry/operator/pkg/config"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	depl "github.com/input-output-hk/catalyst-forge/lib/project/deployment/deployer"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
)

// ReleaseDeploymentReconciler reconciles a Release object
type ReleaseDeploymentReconciler struct {
	client.Client
	ApiClient     api.Client
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

func (r *ReleaseDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 1. Fetch the ReleaseDeployment resource
	var resource foundryv1alpha1.ReleaseDeployment
	if err := r.Get(ctx, req.NamespacedName, &resource); err != nil {
		log.Error(err, "unable to fetch Release")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Fetch the associated ReleaseDeployment from the API
	log.Info("Fetching release deployment from API", "releaseID", resource.Spec.ReleaseID, "deploymentID", resource.Spec.ID)
	releaseDeployment, err := r.ApiClient.GetDeployment(ctx, resource.Spec.ReleaseID, resource.Spec.ID)
	release := releaseDeployment.Release
	if err != nil {
		log.Error(err, "unable to find deployment")
		return ctrl.Result{}, err
	}

	// 3. Check if the deployment has already been completed
	if isDeploymentComplete(releaseDeployment.Status) {
		log.Info("Deployment already succeeded")
		return ctrl.Result{}, nil
	}

	// 4. Set deployment status to running if not already set
	log.Info("Checking deployment status", "status", releaseDeployment.Status)
	if releaseDeployment.Status != api.DeploymentStatusRunning {
		if err := r.ApiClient.UpdateDeploymentStatus(
			ctx,
			release.ID,
			releaseDeployment.ID,
			api.DeploymentStatusRunning,
			"Deployment in progress",
		); err != nil {
			log.Error(err, "unable to update deployment")
			return ctrl.Result{}, err
		}
	}

	// resource.Status.State = "Deploying"
	// if err := r.Status().Update(ctx, &resource); err != nil {
	// 	log.Error(err, "unable to update Release status")
	// 	return ctrl.Result{}, err
	// }

	// 5. Open the deployment repo
	log.Info("Opening deployment repo", "url", r.Config.Deployer.Git.Url)
	deployRepo, err := r.getDeployRepo()
	if err != nil {
		log.Error(err, "unable to get deployment repo")
		return ctrl.Result{}, err
	}

	// 6. Open the source repo
	log.Info("Opening source repo", "url", release.SourceRepo)
	sourceRepo, err := r.getSourceRepo(release)
	if err != nil {
		log.Error(err, "unable to create repo")
		return ctrl.Result{}, err
	}

	// 7. Fetch the bundle from the source repo
	log.Info("Fetching bundle from source repo", "url", release.SourceRepo, "commit", release.SourceCommit)
	bundle, err := deployment.FetchBundle(sourceRepo, release.ProjectPath, r.SecretStore, r.Logger)
	if err != nil {
		log.Error(err, "unable to fetch bundle")
		return ctrl.Result{}, err
	}

	// 8. Create the deployment
	log.Info("Creating deployment", "project", release.Project)
	dp := depl.NewDeployer(
		r.Config.Deployer,
		r.ManifestStore,
		r.SecretStore,
		r.Logger,
		cuecontext.New(),
		depl.WithGitRemoteInteractor(r.Remote),
	)
	deployment, err := dp.CreateDeployment(
		resource.Spec.ID,
		release.Project,
		bundle,
		depl.WithRepo(&deployRepo),
	)
	if err != nil {
		log.Error(err, "unable to create deployment")
		return ctrl.Result{}, err
	}

	// 9. Commit and push the deployment
	log.Info("Committing and pushing deployment")
	if err := deployment.Commit(); err != nil {
		log.Error(err, "unable to commit deployment")
		return ctrl.Result{}, err
	}

	// 10. Update the deployment status to succeeded
	log.Info("Updating deployment status to succeeded")
	if err := r.ApiClient.UpdateDeploymentStatus(
		ctx,
		release.ID,
		releaseDeployment.ID,
		api.DeploymentStatusSucceeded,
		"Deployment succeeded",
	); err != nil {
		log.Error(err, "unable to update deployment")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ReleaseDeploymentReconciler) getDeployRepo() (repo.GitRepo, error) {
	var dfs *billy.BillyFs
	if r.FsDeploy == nil {
		dfs = billy.NewInMemoryFs()
	} else {
		dfs = r.FsDeploy
	}

	rp, err := repo.NewCachedRepo(
		r.Config.Deployer.Git.Url,
		r.Logger,
		repo.WithFS(dfs),
		repo.WithGitRemoteInteractor(r.Remote),
	)
	if err != nil {
		return repo.GitRepo{}, err
	}

	if err := rp.CheckoutBranch(r.Config.Deployer.Git.Ref); err != nil {
		return repo.GitRepo{}, fmt.Errorf("failed to checkout branch: %w", err)
	}

	if err := rp.Pull(); err != nil {
		return repo.GitRepo{}, fmt.Errorf("failed to pull latest changes: %w", err)
	}

	return rp, nil
}

func (r *ReleaseDeploymentReconciler) getSourceRepo(release *api.Release) (repo.GitRepo, error) {
	var sfs *billy.BillyFs
	if r.FsSource == nil {
		sfs = billy.NewBaseOsFS()
	} else {
		sfs = r.FsSource
	}

	rp, err := repo.NewCachedRepo(
		release.SourceRepo,
		r.Logger,
		repo.WithFS(sfs),
		repo.WithGitRemoteInteractor(r.Remote),
	)
	if err != nil {
		return repo.GitRepo{}, err
	}

	if err := rp.Fetch(); err != nil {
		return repo.GitRepo{}, fmt.Errorf("failed to fetch latest changes: %w", err)
	}

	if err := rp.CheckoutCommit(release.SourceCommit); err != nil {
		return repo.GitRepo{}, fmt.Errorf("failed to checkout commit: %w", err)
	}

	return rp, nil
}

// isDeploymentComplete checks if the deployment is complete
// by checking if the status is either Succeeded or Failed.
func isDeploymentComplete(status api.DeploymentStatus) bool {
	return status == api.DeploymentStatusSucceeded || status == api.DeploymentStatusFailed
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&foundryv1alpha1.ReleaseDeployment{}).
		Named("release_deployment").
		Complete(r)
}

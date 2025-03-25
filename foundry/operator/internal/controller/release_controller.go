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

	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
	"github.com/input-output-hk/catalyst-forge/foundry/operator/pkg/config"
	"github.com/input-output-hk/catalyst-forge/foundry/operator/pkg/handlers"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	depl "github.com/input-output-hk/catalyst-forge/lib/project/deployment/deployer"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo/remote"
)

// ReleaseDeploymentReconciler reconciles a Release object
type ReleaseDeploymentReconciler struct {
	client.Client
	Config            config.OperatorConfig
	DeploymentHandler *handlers.ReleaseDeploymentHandler
	Logger            *slog.Logger
	ManifestStore     deployment.ManifestGeneratorStore
	Remote            remote.GitRemoteInteractor
	RepoHandler       *handlers.RepoHandler
	Scheme            *runtime.Scheme
	SecretStore       secrets.SecretStore
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
	if err := r.DeploymentHandler.Load(&resource); err != nil {
		log.Error(err, "unable to load deployment")
		return ctrl.Result{}, err
	}
	release := r.DeploymentHandler.Release()

	// 3. Check if the deployment has already been completed
	if r.DeploymentHandler.IsCompleted() {
		log.Info("Deployment already finished")
		return ctrl.Result{}, nil
	}

	// 4. Check if max attempts have been reached
	if r.DeploymentHandler.MaxAttemptsReached(r.Config.MaxAttempts - 1) {
		log.Info("Max attempts reached, setting deployment to failed")
		if err := r.DeploymentHandler.SetFailed("Max attempts reached"); err != nil {
			log.Error(err, "unable to set deployment status to failed")
			r.DeploymentHandler.AddErrorEvent(nil, "Max attempts reached")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// 5. Set deployment status to running if not already set
	if err := r.DeploymentHandler.SetRunning(); err != nil {
		log.Error(err, "unable to set deployment status to running")
		return ctrl.Result{}, err
	}

	// resource.Status.State = "Deploying"
	// if err := r.Status().Update(ctx, &resource); err != nil {
	// 	log.Error(err, "unable to update Release status")
	// 	return ctrl.Result{}, err
	// }

	// 6. Open repos
	log.Info("Opening deployment repo", "url", r.Config.Deployer.Git.Url)
	if err := r.RepoHandler.LoadDeploymentRepo(r.Config.Deployer.Git.Url, r.Config.Deployer.Git.Ref); err != nil {
		log.Error(err, "unable to load deployment repo")
		return ctrl.Result{}, err
	}

	log.Info("Opening source repo", "url", release.SourceRepo)
	if err := r.RepoHandler.LoadSourceRepo(release.SourceRepo, release.SourceCommit); err != nil {
		log.Error(err, "unable to load source repo")
		r.DeploymentHandler.AddErrorEvent(err, "Unable to load source repo")
		return ctrl.Result{}, err
	}

	// 7. Fetch the bundle from the source repo
	log.Info("Fetching bundle from source repo", "url", release.SourceRepo, "commit", release.SourceCommit)
	bundle, err := deployment.FetchBundle(*r.RepoHandler.SourceRepo(), release.ProjectPath, r.SecretStore, r.Logger)
	if err != nil {
		log.Error(err, "unable to fetch bundle")
		r.DeploymentHandler.AddErrorEvent(err, "Unable to fetch deployment bundle")
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
		depl.WithRepo(r.RepoHandler.DeploymentRepo()),
	)
	if err != nil {
		log.Error(err, "unable to create deployment")
		r.DeploymentHandler.AddErrorEvent(err, "Unable to create deployment")
		return ctrl.Result{}, err
	}

	// 9. Commit and push the deployment
	log.Info("Committing and pushing deployment")
	if err := deployment.Commit(); err != nil {
		log.Error(err, "unable to commit deployment")
		r.DeploymentHandler.AddErrorEvent(err, "Unable to commit deployment")
		return ctrl.Result{}, err
	}

	// 10. Update the deployment status to succeeded
	log.Info("Deployment succeeded")
	if err := r.DeploymentHandler.SetSucceeded(); err != nil {
		log.Error(err, "unable to set deployment status to succeeded")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&foundryv1alpha1.ReleaseDeployment{}).
		Named("release_deployment").
		Complete(r)
}

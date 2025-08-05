package handlers

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/deployments"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/releases"
	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
	"github.com/input-output-hk/catalyst-forge/foundry/operator/pkg/util"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ReleaseDeploymentHandler provides an interface to manage ReleaseDeployment resources.
type ReleaseDeploymentHandler struct {
	ctx        context.Context
	client     client.Client    // API client
	k8sClient  k8sclient.Client // Kubernetes client
	deployment *deployments.ReleaseDeployment
	resource   *foundryv1alpha1.ReleaseDeployment
	clock      util.Clock // Clock interface for time operations
}

// AddErrorEvent adds an error event to the ReleaseDeployment.
func (r *ReleaseDeploymentHandler) AddErrorEvent(err error, reason string) error {
	if err != nil {
		return r.addEvent("DeploymentError", fmt.Sprintf("%s: %s", reason, err.Error()))
	} else {
		return r.addEvent("DeploymentError", reason)
	}
}

// IsCompleted checks if the ReleaseDeployment is completed.
func (r *ReleaseDeploymentHandler) IsCompleted() bool {
	return r.deployment.Status == deployments.DeploymentStatusSucceeded ||
		r.deployment.Status == deployments.DeploymentStatusFailed
}

// IsExpired checks if the deployment has expired based on the TTL.
// It returns true if the CompletionTime plus TTL is before now.
// It also returns the time until expiry.
func (r *ReleaseDeploymentHandler) IsExpired() (bool, time.Duration) {
	if r.resource.Status.CompletionTime == nil {
		return false, 0
	}

	ttlDuration := time.Duration(r.resource.Spec.TTL) * time.Second
	expirationTime := r.resource.Status.CompletionTime.Add(ttlDuration)
	timeUntilExpiry := r.clock.Until(expirationTime)
	return timeUntilExpiry <= 0, timeUntilExpiry
}

// Load loads the ReleaseDeployment from the API and sets it in the handler.
// It returns an error if the deployment cannot be loaded.
func (r *ReleaseDeploymentHandler) Load(resource *foundryv1alpha1.ReleaseDeployment) error {
	r.resource = resource
	releaseDeployment, err := r.client.Deployments().Get(r.ctx, r.resource.Spec.ReleaseID, r.resource.Spec.ID)
	if err != nil {
		return err
	} else if releaseDeployment == nil {
		return fmt.Errorf("unable to find deployment with ID %s", r.resource.Spec.ID)
	}

	r.deployment = releaseDeployment
	return nil
}

// MaxAttemptsReached checks if the maximum number of attempts has been reached.
func (r *ReleaseDeploymentHandler) MaxAttemptsReached(max int) bool {
	return r.deployment.Attempts >= max
}

// Release returns the ReleaseDeployment from the handler.
func (r *ReleaseDeploymentHandler) Release() *releases.Release {
	return r.deployment.Release
}

// SetFailed sets the status of the ReleaseDeployment to failed
func (r *ReleaseDeploymentHandler) SetFailed(reason string) error {
	if err := r.addEvent("DeploymentFailed", reason); err != nil {
		return err
	}

	if r.deployment.Status != deployments.DeploymentStatusFailed {
		if err := r.setAPIStatus(deployments.DeploymentStatusFailed, reason); err != nil {
			return fmt.Errorf("failed to set deployment API status: %w", err)
		}
	}

	if r.resource.Status.State != string(deployments.DeploymentStatusFailed) {
		if err := r.setState(string(deployments.DeploymentStatusFailed)); err != nil {
			return fmt.Errorf("failed to set deployment state: %w", err)
		}
	}

	if err := r.UpdateCompletionTime(); err != nil {
		return fmt.Errorf("failed to update completion time: %w", err)
	}

	return nil
}

// SetRunning sets the status of the ReleaseDeployment to running and increments
// the attempts counter.
func (r *ReleaseDeploymentHandler) SetRunning() error {
	_, err := r.client.Deployments().IncrementAttempts(
		r.ctx,
		r.deployment.ReleaseID,
		r.deployment.ID,
	)
	if err != nil {
		return err
	}

	if err := r.addEvent("DeploymentStarted", "Deployment has started"); err != nil {
		return err
	}

	if r.deployment.Status != deployments.DeploymentStatusRunning {
		return r.setAPIStatus(deployments.DeploymentStatusRunning, "Deployment in progress")
	}

	return nil
}

// SetSucceeded sets the status of the ReleaseDeployment to succeeded.
func (r *ReleaseDeploymentHandler) SetSucceeded() error {
	if err := r.addEvent("DeploymentSucceeded", "Deployment has succeeded"); err != nil {
		return err
	}

	if r.deployment.Status != deployments.DeploymentStatusSucceeded {
		if err := r.setAPIStatus(deployments.DeploymentStatusSucceeded, "Deployment succeeded"); err != nil {
			return fmt.Errorf("failed to set deployment API status: %w", err)
		}
	}

	if r.resource.Status.State != string(deployments.DeploymentStatusSucceeded) {
		if err := r.setState(string(deployments.DeploymentStatusSucceeded)); err != nil {
			return fmt.Errorf("failed to set deployment state: %w", err)
		}
	}

	if err := r.UpdateCompletionTime(); err != nil {
		return fmt.Errorf("failed to update completion time: %w", err)
	}

	return nil
}

// UpdateCompletionTime sets the completion time for the deployment if not already set
// and updates the Kubernetes resource.
func (r *ReleaseDeploymentHandler) UpdateCompletionTime() error {
	if r.resource.Status.CompletionTime == nil {
		now := metav1.NewTime(r.clock.Now())
		r.resource.Status.CompletionTime = &now

		if err := r.k8sClient.Status().Update(context.Background(), r.resource); err != nil {
			return fmt.Errorf("failed to update completion time: %w", err)
		}
	}

	return nil
}

// addEvent adds an event to the ReleaseDeployment.
func (r *ReleaseDeploymentHandler) addEvent(name string, message string) error {
	_, err := r.client.Events().Add(
		r.ctx,
		r.deployment.ReleaseID,
		r.deployment.ID,
		name,
		message,
	)

	return err
}

// setStatus sets the status of the ReleaseDeployment.
func (r *ReleaseDeploymentHandler) setAPIStatus(status deployments.DeploymentStatus, reason string) error {
	r.deployment.Status = status
	r.deployment.Reason = reason
	_, err := r.client.Deployments().Update(
		r.ctx,
		r.deployment.ReleaseID,
		r.deployment,
	)

	return err
}

// setState sets the state of the ReleaseDeployment in Kubernetes.
func (r *ReleaseDeploymentHandler) setState(state string) error {
	r.resource.Status.State = state
	return r.k8sClient.Status().Update(context.Background(), r.resource)
}

// NewReleaseDeploymentHandler creates a new ReleaseDeploymentHandler
func NewReleaseDeploymentHandler(
	ctx context.Context,
	apiClient client.Client,
	k8sClient k8sclient.Client,
) *ReleaseDeploymentHandler {
	return &ReleaseDeploymentHandler{
		ctx:       ctx,
		client:    apiClient,
		k8sClient: k8sClient,
		clock:     util.RealClock{},
	}
}

// NewReleaseDeploymentHandlerWithClock creates a new ReleaseDeploymentHandler with a custom clock
func NewReleaseDeploymentHandlerWithClock(
	ctx context.Context,
	apiClient client.Client,
	k8sClient k8sclient.Client,
	clock util.Clock,
) *ReleaseDeploymentHandler {
	return &ReleaseDeploymentHandler{
		ctx:       ctx,
		client:    apiClient,
		k8sClient: k8sClient,
		clock:     clock,
	}
}

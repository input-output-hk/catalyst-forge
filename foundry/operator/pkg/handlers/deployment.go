package handlers

import (
	"context"
	"fmt"

	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
	foundryv1alpha1 "github.com/input-output-hk/catalyst-forge/foundry/operator/api/v1alpha1"
)

// ReleaseDeploymentHandler provides an interface to manage ReleaseDeployment resources.
type ReleaseDeploymentHandler struct {
	ctx        context.Context
	client     client.Client
	deployment *client.ReleaseDeployment
	resource   *foundryv1alpha1.ReleaseDeployment
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
	return r.deployment.Status == client.DeploymentStatusSucceeded ||
		r.deployment.Status == client.DeploymentStatusFailed
}

// Load loads the ReleaseDeployment from the API and sets it in the handler.
// It returns an error if the deployment cannot be loaded.
func (r *ReleaseDeploymentHandler) Load(resource *foundryv1alpha1.ReleaseDeployment) error {
	r.resource = resource
	releaseDeployment, err := r.client.GetDeployment(r.ctx, r.resource.Spec.ReleaseID, r.resource.Spec.ID)
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
func (r *ReleaseDeploymentHandler) Release() *client.Release {
	return r.deployment.Release
}

// SetFailed sets the status of the ReleaseDeployment to failed
func (r *ReleaseDeploymentHandler) SetFailed(reason string) error {
	if err := r.addEvent("DeploymentFailed", reason); err != nil {
		return err
	}

	if r.deployment.Status != client.DeploymentStatusFailed {
		return r.setStatus(client.DeploymentStatusFailed, reason)
	}

	return nil
}

// SetRunning sets the status of the ReleaseDeployment to running and increments
// the attempts counter.
func (r *ReleaseDeploymentHandler) SetRunning() error {
	_, err := r.client.IncrementDeploymentAttempts(
		r.ctx,
		r.deployment.Release.ID,
		r.deployment.ID,
	)
	if err != nil {
		return err
	}

	if err := r.addEvent("DeploymentStarted", "Deployment has started"); err != nil {
		return err
	}

	if r.deployment.Status != client.DeploymentStatusRunning {
		return r.setStatus(client.DeploymentStatusRunning, "Deployment in progress")
	}

	return nil
}

// SetFailed sets the status of the ReleaseDeployment to failed.
func (r *ReleaseDeploymentHandler) SetSucceeded() error {
	if err := r.addEvent("DeploymentSucceeded", "Deployment has succeeded"); err != nil {
		return err
	}

	if r.deployment.Status != client.DeploymentStatusSucceeded {
		return r.setStatus(client.DeploymentStatusSucceeded, "Deployment succeeded")
	}

	return nil
}

// addEvent adds an event to the ReleaseDeployment.
func (r *ReleaseDeploymentHandler) addEvent(name string, message string) error {
	_, err := r.client.AddDeploymentEvent(
		r.ctx,
		r.deployment.ReleaseID,
		r.deployment.ID,
		name,
		message,
	)

	return err
}

// setStatus sets the status of the ReleaseDeployment.
func (r *ReleaseDeploymentHandler) setStatus(status client.DeploymentStatus, reason string) error {
	r.deployment.Status = status
	r.deployment.Reason = reason
	_, err := r.client.UpdateDeployment(
		r.ctx,
		r.deployment.Release.ID,
		r.deployment,
	)

	return err
}

// NewReleaseDeploymentHandler creates a new ReleaseDeploymentHandler
func NewReleaseDeploymentHandler(
	ctx context.Context,
	client client.Client,
) *ReleaseDeploymentHandler {
	return &ReleaseDeploymentHandler{
		ctx:    ctx,
		client: client,
	}
}

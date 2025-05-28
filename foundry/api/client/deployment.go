package client

import (
	"context"
	"fmt"
	"net/http"
)

// CreateDeployment creates a new deployment for a release
func (c *HTTPClient) CreateDeployment(ctx context.Context, releaseID string) (*ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deploy", releaseID)

	var resp ReleaseDeployment
	err := c.do(ctx, http.MethodPost, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetDeployment retrieves a specific deployment
func (c *HTTPClient) GetDeployment(ctx context.Context, releaseID string, deployID string) (*ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deploy/%s", releaseID, deployID)

	var resp ReleaseDeployment
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateDeployment updates a deployment with new values
func (c *HTTPClient) UpdateDeployment(ctx context.Context, releaseID string, deployment *ReleaseDeployment) (*ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deploy/%s", releaseID, deployment.ID)

	var resp ReleaseDeployment
	err := c.do(ctx, http.MethodPut, path, deployment, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListDeployments retrieves all deployments for a release
func (c *HTTPClient) ListDeployments(ctx context.Context, releaseID string) ([]ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deployments", releaseID)

	var resp []ReleaseDeployment
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// IncrementDeploymentAttempts increments the attempts counter for a deployment by 1
func (c *HTTPClient) IncrementDeploymentAttempts(ctx context.Context, releaseID string, deployID string) (*ReleaseDeployment, error) {
	// First get the current deployment
	deployment, err := c.GetDeployment(ctx, releaseID, deployID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Increment the attempts counter
	deployment.Attempts++

	// Update the deployment
	return c.UpdateDeployment(ctx, releaseID, deployment)
}

// GetLatestDeployment retrieves the most recent deployment for a release
func (c *HTTPClient) GetLatestDeployment(ctx context.Context, releaseID string) (*ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deploy/latest", releaseID)

	var resp ReleaseDeployment
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

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

// UpdateDeploymentStatus updates the status of a deployment
func (c *HTTPClient) UpdateDeploymentStatus(ctx context.Context, releaseID string, deployID string, status DeploymentStatus, reason string) error {
	path := fmt.Sprintf("/release/%s/deploy/%s/status", releaseID, deployID)

	payload := struct {
		Status DeploymentStatus `json:"status"`
		Reason string           `json:"reason"`
	}{
		Status: status,
		Reason: reason,
	}

	return c.do(ctx, http.MethodPut, path, payload, nil)
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

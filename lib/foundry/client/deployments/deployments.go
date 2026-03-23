package deployments

import (
	"context"
	"fmt"
	"net/http"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/deployments.go . DeploymentsClientInterface

// DeploymentsClientInterface defines the interface for deployment operations
type DeploymentsClientInterface interface {
	Create(ctx context.Context, releaseID string) (*ReleaseDeployment, error)
	Get(ctx context.Context, releaseID string, deployID string) (*ReleaseDeployment, error)
	Update(ctx context.Context, releaseID string, deployment *ReleaseDeployment) (*ReleaseDeployment, error)
	List(ctx context.Context, releaseID string) ([]ReleaseDeployment, error)
	IncrementAttempts(ctx context.Context, releaseID string, deployID string) (*ReleaseDeployment, error)
	GetLatest(ctx context.Context, releaseID string) (*ReleaseDeployment, error)
}

// DeploymentsClient handles deployment-related operations
type DeploymentsClient struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

// Ensure DeploymentsClient implements DeploymentsClientInterface
var _ DeploymentsClientInterface = (*DeploymentsClient)(nil)

// NewDeploymentsClient creates a new deployments client
func NewDeploymentsClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *DeploymentsClient {
	return &DeploymentsClient{do: do}
}

// Create creates a new deployment for a release
func (c *DeploymentsClient) Create(ctx context.Context, releaseID string) (*ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deploy", releaseID)

	var resp ReleaseDeployment
	err := c.do(ctx, http.MethodPost, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Get retrieves a specific deployment
func (c *DeploymentsClient) Get(ctx context.Context, releaseID string, deployID string) (*ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deploy/%s", releaseID, deployID)

	var resp ReleaseDeployment
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Update updates a deployment with new values
func (c *DeploymentsClient) Update(ctx context.Context, releaseID string, deployment *ReleaseDeployment) (*ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deploy/%s", releaseID, deployment.ID)

	var resp ReleaseDeployment
	err := c.do(ctx, http.MethodPut, path, deployment, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// List retrieves all deployments for a release
func (c *DeploymentsClient) List(ctx context.Context, releaseID string) ([]ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deployments", releaseID)

	var resp []ReleaseDeployment
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// IncrementAttempts increments the attempts counter for a deployment by 1
func (c *DeploymentsClient) IncrementAttempts(ctx context.Context, releaseID string, deployID string) (*ReleaseDeployment, error) {
	// First get the current deployment
	deployment, err := c.Get(ctx, releaseID, deployID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Increment the attempts counter
	deployment.Attempts++

	// Update the deployment
	return c.Update(ctx, releaseID, deployment)
}

// GetLatest retrieves the most recent deployment for a release
func (c *DeploymentsClient) GetLatest(ctx context.Context, releaseID string) (*ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deploy/latest", releaseID)

	var resp ReleaseDeployment
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

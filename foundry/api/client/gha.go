package client

import (
	"context"
	"fmt"
	"net/http"
)

// ValidateToken validates a GitHub Actions token and returns a JWT token
func (c *HTTPClient) ValidateToken(ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, error) {
	var resp ValidateTokenResponse
	err := c.do(ctx, http.MethodPost, "/gha/validate", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateAuth creates a new GHA authentication configuration
func (c *HTTPClient) CreateAuth(ctx context.Context, req *CreateAuthRequest) (*GHARepositoryAuth, error) {
	var resp GHARepositoryAuth
	err := c.do(ctx, http.MethodPost, "/gha/auth", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAuth retrieves a GHA authentication configuration by ID
func (c *HTTPClient) GetAuth(ctx context.Context, id uint) (*GHARepositoryAuth, error) {
	path := fmt.Sprintf("/gha/auth/%d", id)

	var resp GHARepositoryAuth
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAuthByRepository retrieves a GHA authentication configuration by repository
func (c *HTTPClient) GetAuthByRepository(ctx context.Context, repository string) (*GHARepositoryAuth, error) {
	path := fmt.Sprintf("/gha/auth/repository/%s", repository)

	var resp GHARepositoryAuth
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateAuth updates a GHA authentication configuration
func (c *HTTPClient) UpdateAuth(ctx context.Context, id uint, req *UpdateAuthRequest) (*GHARepositoryAuth, error) {
	path := fmt.Sprintf("/gha/auth/%d", id)

	var resp GHARepositoryAuth
	err := c.do(ctx, http.MethodPut, path, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteAuth deletes a GHA authentication configuration
func (c *HTTPClient) DeleteAuth(ctx context.Context, id uint) error {
	path := fmt.Sprintf("/gha/auth/%d", id)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ListAuths retrieves all GHA authentication configurations
func (c *HTTPClient) ListAuths(ctx context.Context) ([]GHARepositoryAuth, error) {
	var resp []GHARepositoryAuth
	err := c.do(ctx, http.MethodGet, "/gha/auth", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

package gha

import (
	"context"
	"fmt"
	"net/http"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/gha.go . GHAClientInterface

// GHAClientInterface defines the interface for GitHub Actions operations
type GHAClientInterface interface {
	ValidateToken(ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, error)
	CreateAuth(ctx context.Context, req *CreateAuthRequest) (*GHARepositoryAuth, error)
	GetAuth(ctx context.Context, id uint) (*GHARepositoryAuth, error)
	GetAuthByRepository(ctx context.Context, repository string) (*GHARepositoryAuth, error)
	UpdateAuth(ctx context.Context, id uint, req *UpdateAuthRequest) (*GHARepositoryAuth, error)
	DeleteAuth(ctx context.Context, id uint) error
	ListAuths(ctx context.Context) ([]GHARepositoryAuth, error)
}

// GHAClient handles GitHub Actions-related operations
type GHAClient struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

// Ensure GHAClient implements GHAClientInterface
var _ GHAClientInterface = (*GHAClient)(nil)

// NewGHAClient creates a new GHA client
func NewGHAClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *GHAClient {
	return &GHAClient{do: do}
}

// ValidateToken validates a GitHub Actions token and returns a JWT token
func (c *GHAClient) ValidateToken(ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, error) {
	var resp ValidateTokenResponse
	err := c.do(ctx, http.MethodPost, "/gha/validate", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateAuth creates a new GHA authentication configuration
func (c *GHAClient) CreateAuth(ctx context.Context, req *CreateAuthRequest) (*GHARepositoryAuth, error) {
	var resp GHARepositoryAuth
	err := c.do(ctx, http.MethodPost, "/gha/auth", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAuth retrieves a GHA authentication configuration by ID
func (c *GHAClient) GetAuth(ctx context.Context, id uint) (*GHARepositoryAuth, error) {
	path := fmt.Sprintf("/gha/auth/%d", id)

	var resp GHARepositoryAuth
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAuthByRepository retrieves a GHA authentication configuration by repository
func (c *GHAClient) GetAuthByRepository(ctx context.Context, repository string) (*GHARepositoryAuth, error) {
	path := fmt.Sprintf("/gha/auth/repository/%s", repository)

	var resp GHARepositoryAuth
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateAuth updates a GHA authentication configuration
func (c *GHAClient) UpdateAuth(ctx context.Context, id uint, req *UpdateAuthRequest) (*GHARepositoryAuth, error) {
	path := fmt.Sprintf("/gha/auth/%d", id)

	var resp GHARepositoryAuth
	err := c.do(ctx, http.MethodPut, path, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteAuth deletes a GHA authentication configuration
func (c *GHAClient) DeleteAuth(ctx context.Context, id uint) error {
	path := fmt.Sprintf("/gha/auth/%d", id)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ListAuths retrieves all GHA authentication configurations
func (c *GHAClient) ListAuths(ctx context.Context) ([]GHARepositoryAuth, error) {
	var resp []GHARepositoryAuth
	err := c.do(ctx, http.MethodGet, "/gha/auth", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

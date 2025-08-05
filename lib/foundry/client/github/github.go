package github

import (
	"context"
	"fmt"
	"net/http"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/github.go . GithubClientInterface

// GithubClientInterface defines the interface for GitHub Actions operations
type GithubClientInterface interface {
	ValidateToken(ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, error)
	CreateAuth(ctx context.Context, req *CreateAuthRequest) (*GithubRepositoryAuth, error)
	GetAuth(ctx context.Context, id uint) (*GithubRepositoryAuth, error)
	GetAuthByRepository(ctx context.Context, repository string) (*GithubRepositoryAuth, error)
	UpdateAuth(ctx context.Context, id uint, req *UpdateAuthRequest) (*GithubRepositoryAuth, error)
	DeleteAuth(ctx context.Context, id uint) error
	ListAuths(ctx context.Context) ([]GithubRepositoryAuth, error)
}

// GithubClient handles GitHub Actions-related operations
type GithubClient struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

// Ensure GithubClient implements GithubClientInterface
var _ GithubClientInterface = (*GithubClient)(nil)

// NewGithubClient creates a new GitHub client
func NewGithubClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *GithubClient {
	return &GithubClient{do: do}
}

// ValidateToken validates a GitHub Actions token and returns a JWT token
func (c *GithubClient) ValidateToken(ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, error) {
	var resp ValidateTokenResponse
	err := c.do(ctx, http.MethodPost, "/auth/github/login", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateAuth creates a new GHA authentication configuration
func (c *GithubClient) CreateAuth(ctx context.Context, req *CreateAuthRequest) (*GithubRepositoryAuth, error) {
	var resp GithubRepositoryAuth
	err := c.do(ctx, http.MethodPost, "/auth/github", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAuth retrieves a GHA authentication configuration by ID
func (c *GithubClient) GetAuth(ctx context.Context, id uint) (*GithubRepositoryAuth, error) {
	path := fmt.Sprintf("/auth/github/%d", id)

	var resp GithubRepositoryAuth
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAuthByRepository retrieves a GHA authentication configuration by repository
func (c *GithubClient) GetAuthByRepository(ctx context.Context, repository string) (*GithubRepositoryAuth, error) {
	path := fmt.Sprintf("/auth/github/repository/%s", repository)

	var resp GithubRepositoryAuth
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateAuth updates a GHA authentication configuration
func (c *GithubClient) UpdateAuth(ctx context.Context, id uint, req *UpdateAuthRequest) (*GithubRepositoryAuth, error) {
	path := fmt.Sprintf("/auth/github/%d", id)

	var resp GithubRepositoryAuth
	err := c.do(ctx, http.MethodPut, path, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteAuth deletes a GHA authentication configuration
func (c *GithubClient) DeleteAuth(ctx context.Context, id uint) error {
	path := fmt.Sprintf("/auth/github/%d", id)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ListAuths retrieves all GHA authentication configurations
func (c *GithubClient) ListAuths(ctx context.Context) ([]GithubRepositoryAuth, error) {
	var resp []GithubRepositoryAuth
	err := c.do(ctx, http.MethodGet, "/auth/github", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

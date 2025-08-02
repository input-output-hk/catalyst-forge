package releases

import (
	"context"
	"fmt"
	"net/http"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/releases.go . ReleasesClientInterface

// ReleasesClientInterface defines the interface for release operations
type ReleasesClientInterface interface {
	Create(ctx context.Context, release *Release, deploy bool) (*Release, error)
	Get(ctx context.Context, id string) (*Release, error)
	Update(ctx context.Context, release *Release) (*Release, error)
	List(ctx context.Context, projectName string) ([]Release, error)
	GetByAlias(ctx context.Context, aliasName string) (*Release, error)
}

// ReleasesClient handles release-related operations
type ReleasesClient struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

// Ensure ReleasesClient implements ReleasesClientInterface
var _ ReleasesClientInterface = (*ReleasesClient)(nil)

// NewReleasesClient creates a new releases client
func NewReleasesClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *ReleasesClient {
	return &ReleasesClient{do: do}
}

// Create creates a new release
func (c *ReleasesClient) Create(ctx context.Context, release *Release, deploy bool) (*Release, error) {
	path := "/release"
	if deploy {
		path += "?deploy=true"
	}

	var resp Release
	err := c.do(ctx, http.MethodPost, path, release, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Get retrieves a release by ID
func (c *ReleasesClient) Get(ctx context.Context, id string) (*Release, error) {
	path := fmt.Sprintf("/release/%s", id)

	var resp Release
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Update updates an existing release
func (c *ReleasesClient) Update(ctx context.Context, release *Release) (*Release, error) {
	path := fmt.Sprintf("/release/%s", release.ID)

	var resp Release
	err := c.do(ctx, http.MethodPut, path, release, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// List retrieves all releases for a project
func (c *ReleasesClient) List(ctx context.Context, projectName string) ([]Release, error) {
	path := "/releases"
	if projectName != "" {
		path += fmt.Sprintf("?project=%s", projectName)
	}

	var resp []Release
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetByAlias retrieves a release by its alias
func (c *ReleasesClient) GetByAlias(ctx context.Context, aliasName string) (*Release, error) {
	path := fmt.Sprintf("/release/alias/%s", aliasName)

	var resp Release
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

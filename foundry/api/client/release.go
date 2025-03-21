package client

import (
	"context"
	"fmt"
	"net/http"
)

// CreateRelease creates a new release
func (c *HTTPClient) CreateRelease(ctx context.Context, release *Release, deploy bool) (*Release, error) {
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

// GetRelease retrieves a release by ID
func (c *HTTPClient) GetRelease(ctx context.Context, id string) (*Release, error) {
	path := fmt.Sprintf("/release/%s", id)

	var resp Release
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// UpdateRelease updates an existing release
func (c *HTTPClient) UpdateRelease(ctx context.Context, release *Release) (*Release, error) {
	path := fmt.Sprintf("/release/%s", release.ID)

	var resp Release
	err := c.do(ctx, http.MethodPut, path, release, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListReleases retrieves all releases for a project
func (c *HTTPClient) ListReleases(ctx context.Context, projectName string) ([]Release, error) {
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

// GetReleaseByAlias retrieves a release by its alias
func (c *HTTPClient) GetReleaseByAlias(ctx context.Context, aliasName string) (*Release, error) {
	path := fmt.Sprintf("/release/alias/%s", aliasName)

	var resp Release
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

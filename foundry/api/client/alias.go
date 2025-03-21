package client

import (
	"context"
	"fmt"
	"net/http"
)

// CreateAlias creates a new alias for a release
func (c *HTTPClient) CreateAlias(ctx context.Context, aliasName string, releaseID string) error {
	path := fmt.Sprintf("/release/alias/%s", aliasName)

	payload := struct {
		ReleaseID string `json:"release_id"`
	}{
		ReleaseID: releaseID,
	}

	return c.do(ctx, http.MethodPost, path, payload, nil)
}

// DeleteAlias removes an alias
func (c *HTTPClient) DeleteAlias(ctx context.Context, aliasName string) error {
	path := fmt.Sprintf("/release/alias/%s", aliasName)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ListAliases retrieves all aliases for a release
func (c *HTTPClient) ListAliases(ctx context.Context, releaseID string) ([]ReleaseAlias, error) {
	path := fmt.Sprintf("/release/%s/aliases", releaseID)

	var resp []ReleaseAlias
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

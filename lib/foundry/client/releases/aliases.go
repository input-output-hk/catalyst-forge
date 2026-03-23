package releases

import (
	"context"
	"fmt"
	"net/http"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/aliases.go . AliasesClientInterface

// AliasesClientInterface defines the interface for release alias operations
type AliasesClientInterface interface {
	Create(ctx context.Context, aliasName string, releaseID string) error
	Delete(ctx context.Context, aliasName string) error
	List(ctx context.Context, releaseID string) ([]ReleaseAlias, error)
}

// AliasesClient handles release alias-related operations
type AliasesClient struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

// Ensure AliasesClient implements AliasesClientInterface
var _ AliasesClientInterface = (*AliasesClient)(nil)

// NewAliasesClient creates a new aliases client
func NewAliasesClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *AliasesClient {
	return &AliasesClient{do: do}
}

// Create creates a new alias for a release
func (c *AliasesClient) Create(ctx context.Context, aliasName string, releaseID string) error {
	path := fmt.Sprintf("/release/alias/%s", aliasName)

	payload := struct {
		ReleaseID string `json:"release_id"`
	}{
		ReleaseID: releaseID,
	}

	return c.do(ctx, http.MethodPost, path, payload, nil)
}

// Delete removes an alias
func (c *AliasesClient) Delete(ctx context.Context, aliasName string) error {
	path := fmt.Sprintf("/release/alias/%s", aliasName)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// List retrieves all aliases for a release
func (c *AliasesClient) List(ctx context.Context, releaseID string) ([]ReleaseAlias, error) {
	path := fmt.Sprintf("/release/%s/aliases", releaseID)

	var resp []ReleaseAlias
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

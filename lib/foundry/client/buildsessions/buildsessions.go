package buildsessions

import (
	"context"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/buildsessions.go . BuildSessionsClientInterface

type BuildSessionsClientInterface interface {
	Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error)
}

type Client struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

func NewBuildSessionsClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *Client {
	return &Client{do: do}
}

type CreateRequest struct {
	OwnerType string                 `json:"owner_type"`
	OwnerID   string                 `json:"owner_id"`
	TTL       string                 `json:"ttl"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type CreateResponse struct {
	ID        string `json:"id"`
	ExpiresAt string `json:"expires_at"`
}

func (c *Client) Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	var out CreateResponse
	if err := c.do(ctx, "POST", "/build/sessions", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

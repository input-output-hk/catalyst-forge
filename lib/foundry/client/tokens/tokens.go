package tokens

import (
	"context"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/tokens.go . TokensClientInterface

type TokensClientInterface interface {
	Refresh(ctx context.Context, req *RefreshRequest) (*RefreshResponse, error)
	Revoke(ctx context.Context, req *RevokeRequest) error
}

type Client struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

func NewTokensClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *Client {
	return &Client{do: do}
}

type RefreshRequest struct {
	Refresh string `json:"refresh"`
}
type RefreshResponse struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}
type RevokeRequest struct {
	Refresh string `json:"refresh"`
}

func (c *Client) Refresh(ctx context.Context, req *RefreshRequest) (*RefreshResponse, error) {
	var out RefreshResponse
	if err := c.do(ctx, "POST", "/tokens/refresh", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Revoke(ctx context.Context, req *RevokeRequest) error {
	return c.do(ctx, "POST", "/tokens/revoke", req, nil)
}

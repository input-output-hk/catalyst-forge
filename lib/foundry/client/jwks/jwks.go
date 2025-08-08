package jwks

import (
	"context"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/jwks.go . JWKSClientInterface

type JWKSClientInterface interface {
	Get(ctx context.Context) ([]byte, error)
}

type Client struct {
	doRaw func(ctx context.Context, method, path string, reqBody interface{}) ([]byte, error)
}

func NewJWKSClient(doRaw func(ctx context.Context, method, path string, reqBody interface{}) ([]byte, error)) *Client {
	return &Client{doRaw: doRaw}
}

func (c *Client) Get(ctx context.Context) ([]byte, error) {
	return c.doRaw(ctx, "GET", "/.well-known/jwks.json", nil)
}

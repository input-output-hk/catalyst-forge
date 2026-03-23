package extauthz

import "context"

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/extauthz.go . ExtAuthzClientInterface

type ExtAuthzClientInterface interface {
	AuthorizeGateway(ctx context.Context, req *AuthorizeRequest) (*AuthorizeResponse, error)
}

type Client struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

func NewExtAuthzClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *Client {
	return &Client{do: do}
}

type AuthorizeRequest struct {
	SAN    string `json:"san"`
	Policy string `json:"policy_prefix,omitempty"`
}

type AuthorizeResponse struct {
	Allowed bool `json:"allowed"`
}

func (c *Client) AuthorizeGateway(ctx context.Context, req *AuthorizeRequest) (*AuthorizeResponse, error) {
	var out AuthorizeResponse
	if err := c.do(ctx, "POST", "/build/gateway/authorize", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

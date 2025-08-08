package invites

import (
	"context"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/invites.go . InvitesClientInterface

type InvitesClientInterface interface {
	Create(ctx context.Context, req *CreateInviteRequest) (*CreateInviteResponse, error)
	Verify(ctx context.Context, token string) error
}

type Client struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

func NewInvitesClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *Client {
	return &Client{do: do}
}

type CreateInviteRequest struct {
	Email string   `json:"email"`
	Roles []string `json:"roles"`
	TTL   string   `json:"ttl,omitempty"`
}
type CreateInviteResponse struct {
	ID    uint   `json:"id"`
	Token string `json:"token"`
}

func (c *Client) Create(ctx context.Context, req *CreateInviteRequest) (*CreateInviteResponse, error) {
	var out CreateInviteResponse
	if err := c.do(ctx, "POST", "/auth/invites", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Verify(ctx context.Context, token string) error {
	path := "/verify?token=" + token
	return c.do(ctx, "GET", path, nil, nil)
}

package device

import (
	"context"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/device.go . DeviceClientInterface

type DeviceClientInterface interface {
	Init(ctx context.Context, req *InitRequest) (*InitResponse, error)
	Token(ctx context.Context, req *TokenRequest) (*TokenResponse, error)
	Approve(ctx context.Context, req *ApproveRequest) error
}

type Client struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

func NewDeviceClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *Client {
	return &Client{do: do}
}

type InitRequest struct {
	Name        string `json:"name,omitempty"`
	Platform    string `json:"platform,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}
type InitResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}
type TokenRequest struct {
	DeviceCode string `json:"device_code"`
}
type TokenResponse struct {
	Access  string `json:"access,omitempty"`
	Refresh string `json:"refresh,omitempty"`
	Error   string `json:"error,omitempty"`
}
type ApproveRequest struct {
	UserCode string `json:"user_code"`
}

func (c *Client) Init(ctx context.Context, req *InitRequest) (*InitResponse, error) {
	var out InitResponse
	if err := c.do(ctx, "POST", "/device/init", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Token(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	var out TokenResponse
	if err := c.do(ctx, "POST", "/device/token", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Approve(ctx context.Context, req *ApproveRequest) error {
	return c.do(ctx, "POST", "/device/approve", req, nil)
}

package auth

import (
	"context"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/auth.go . AuthClientInterface

// AuthClientInterface defines the interface for authentication operations
type AuthClientInterface interface {
	CreateChallenge(ctx context.Context, req *ChallengeRequest) (*ChallengeResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)

	DeviceRegistrationInit(ctx context.Context, req *DeviceRegistrationInitRequest) (*DeviceRegistrationInitResponse, error)
	DeviceRegister(ctx context.Context, req *DeviceRegisterRequest) (*DeviceRegisterResponse, error)
	DeviceRefresh(ctx context.Context, req *DeviceRefreshRequest) (*DeviceRefreshResponse, error)
	DeviceLogout(ctx context.Context) error
	GetDevices(ctx context.Context) ([]DeviceListResponse, error)
	DeleteDevice(ctx context.Context, deviceID string) error
}

// AuthClient handles authentication-related operations
type AuthClient struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

// Ensure AuthClient implements AuthClientInterface
var _ AuthClientInterface = (*AuthClient)(nil)

// NewAuthClient creates a new auth client
func NewAuthClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *AuthClient {
	return &AuthClient{do: do}
}

// CreateChallenge creates a new authentication challenge
func (c *AuthClient) CreateChallenge(ctx context.Context, req *ChallengeRequest) (*ChallengeResponse, error) {
	var challenge ChallengeResponse
	err := c.do(ctx, "POST", "/auth/challenge", req, &challenge)
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

// Login authenticates a user with a challenge response
func (c *AuthClient) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	var response LoginResponse
	err := c.do(ctx, "POST", "/auth/login", req, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// DeviceRegistrationInit initializes device registration flow
func (c *AuthClient) DeviceRegistrationInit(ctx context.Context, req *DeviceRegistrationInitRequest) (*DeviceRegistrationInitResponse, error) {
	var response DeviceRegistrationInitResponse
	err := c.do(ctx, "POST", "/auth/devices/init", req, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// DeviceRegister completes device registration by providing device proof
func (c *AuthClient) DeviceRegister(ctx context.Context, req *DeviceRegisterRequest) (*DeviceRegisterResponse, error) {
	var response DeviceRegisterResponse
	err := c.do(ctx, "POST", "/auth/devices/register", req, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// DeviceRefresh refreshes an access token using device proof and refresh token cookie
func (c *AuthClient) DeviceRefresh(ctx context.Context, req *DeviceRefreshRequest) (*DeviceRefreshResponse, error) {
	var response DeviceRefreshResponse
	err := c.do(ctx, "POST", "/auth/refresh", req, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// DeviceLogout logs out the user and revokes the current refresh token
func (c *AuthClient) DeviceLogout(ctx context.Context) error {
	return c.do(ctx, "POST", "/auth/logout", nil, nil)
}

// GetDevices retrieves the list of devices registered to the authenticated user
func (c *AuthClient) GetDevices(ctx context.Context) ([]DeviceListResponse, error) {
	var devices []DeviceListResponse
	err := c.do(ctx, "GET", "/auth/devices", nil, &devices)
	if err != nil {
		return nil, err
	}
	return devices, nil
}

// DeleteDevice revokes a specific device and invalidates all its refresh tokens
func (c *AuthClient) DeleteDevice(ctx context.Context, deviceID string) error {
	return c.do(ctx, "DELETE", "/auth/devices/"+deviceID, nil, nil)
}

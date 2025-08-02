package auth

import (
	"context"

	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/auth.go . AuthClientInterface

// AuthClientInterface defines the interface for authentication operations
type AuthClientInterface interface {
	CreateChallenge(ctx context.Context, req *ChallengeRequest) (*auth.KeyPairChallenge, error)
	Login(ctx context.Context, req *auth.KeyPairChallengeResponse) (*LoginResponse, error)
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
func (c *AuthClient) CreateChallenge(ctx context.Context, req *ChallengeRequest) (*auth.KeyPairChallenge, error) {
	var challenge auth.KeyPairChallenge
	err := c.do(ctx, "POST", "/auth/challenge", req, &challenge)
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

// Login authenticates a user with a challenge response
func (c *AuthClient) Login(ctx context.Context, req *auth.KeyPairChallengeResponse) (*LoginResponse, error) {
	var response LoginResponse
	err := c.do(ctx, "POST", "/auth/login", req, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

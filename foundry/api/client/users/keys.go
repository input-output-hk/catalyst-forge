package users

import (
	"context"
	"fmt"
)

// KeysClientInterface defines the interface for user key operations
type KeysClientInterface interface {
	Create(ctx context.Context, req *CreateUserKeyRequest) (*UserKey, error)
	Register(ctx context.Context, req *RegisterUserKeyRequest) (*UserKey, error)
	Get(ctx context.Context, id uint) (*UserKey, error)
	GetByKid(ctx context.Context, kid string) (*UserKey, error)
	Update(ctx context.Context, id uint, req *UpdateUserKeyRequest) (*UserKey, error)
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context) ([]UserKey, error)
	Revoke(ctx context.Context, id uint) (*UserKey, error)
	GetByUserID(ctx context.Context, userID uint) ([]UserKey, error)
	GetActiveByUserID(ctx context.Context, userID uint) ([]UserKey, error)
	GetInactiveByUserID(ctx context.Context, userID uint) ([]UserKey, error)
	GetInactive(ctx context.Context) ([]UserKey, error)
}

// KeysClient handles user key-related operations
type KeysClient struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

// Ensure KeysClient implements KeysClientInterface
var _ KeysClientInterface = (*KeysClient)(nil)

// NewKeysClient creates a new keys client
func NewKeysClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *KeysClient {
	return &KeysClient{do: do}
}

// Create creates a new user key
func (c *KeysClient) Create(ctx context.Context, req *CreateUserKeyRequest) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "POST", "/auth/keys", req, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

// Register registers a new user key
func (c *KeysClient) Register(ctx context.Context, req *RegisterUserKeyRequest) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "POST", "/auth/keys/register", req, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

// Get retrieves a user key by ID
func (c *KeysClient) Get(ctx context.Context, id uint) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/keys/%d", id), nil, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

// GetByKid retrieves a user key by key ID
func (c *KeysClient) GetByKid(ctx context.Context, kid string) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/keys/kid/%s", kid), nil, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

// Update updates a user key
func (c *KeysClient) Update(ctx context.Context, id uint, req *UpdateUserKeyRequest) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "PUT", fmt.Sprintf("/auth/keys/%d", id), req, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

// Delete deletes a user key
func (c *KeysClient) Delete(ctx context.Context, id uint) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/auth/keys/%d", id), nil, nil)
}

// List retrieves all user keys
func (c *KeysClient) List(ctx context.Context) ([]UserKey, error) {
	var userKeys []UserKey
	err := c.do(ctx, "GET", "/auth/keys", nil, &userKeys)
	if err != nil {
		return nil, err
	}
	return userKeys, nil
}

// Revoke revokes a user key
func (c *KeysClient) Revoke(ctx context.Context, id uint) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "POST", fmt.Sprintf("/auth/keys/%d/revoke", id), nil, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

// GetByUserID retrieves all user keys for a specific user
func (c *KeysClient) GetByUserID(ctx context.Context, userID uint) ([]UserKey, error) {
	var userKeys []UserKey
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/keys/user/%d", userID), nil, &userKeys)
	if err != nil {
		return nil, err
	}
	return userKeys, nil
}

// GetActiveByUserID retrieves all active user keys for a specific user
func (c *KeysClient) GetActiveByUserID(ctx context.Context, userID uint) ([]UserKey, error) {
	var userKeys []UserKey
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/keys/user/%d/active", userID), nil, &userKeys)
	if err != nil {
		return nil, err
	}
	return userKeys, nil
}

// GetInactiveByUserID retrieves all inactive user keys for a specific user
func (c *KeysClient) GetInactiveByUserID(ctx context.Context, userID uint) ([]UserKey, error) {
	var userKeys []UserKey
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/keys/user/%d/inactive", userID), nil, &userKeys)
	if err != nil {
		return nil, err
	}
	return userKeys, nil
}

// GetInactive retrieves all inactive user keys
func (c *KeysClient) GetInactive(ctx context.Context) ([]UserKey, error) {
	var userKeys []UserKey
	err := c.do(ctx, "GET", "/auth/pending/keys", nil, &userKeys)
	if err != nil {
		return nil, err
	}
	return userKeys, nil
}

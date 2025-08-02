package users

import (
	"context"
	"fmt"
)

// UsersClientInterface defines the interface for user operations
type UsersClientInterface interface {
	Create(ctx context.Context, req *CreateUserRequest) (*User, error)
	Register(ctx context.Context, req *RegisterUserRequest) (*User, error)
	Get(ctx context.Context, id uint) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id uint, req *UpdateUserRequest) (*User, error)
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context) ([]User, error)
	GetPending(ctx context.Context) ([]User, error)
	Activate(ctx context.Context, id uint) (*User, error)
	Deactivate(ctx context.Context, id uint) (*User, error)
}

// UsersClient handles user-related operations
type UsersClient struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

// Ensure UsersClient implements UsersClientInterface
var _ UsersClientInterface = (*UsersClient)(nil)

// NewUsersClient creates a new users client
func NewUsersClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *UsersClient {
	return &UsersClient{do: do}
}

// Create creates a new user
func (c *UsersClient) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
	var user User
	err := c.do(ctx, "POST", "/auth/users", req, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Register registers a new user
func (c *UsersClient) Register(ctx context.Context, req *RegisterUserRequest) (*User, error) {
	var user User
	err := c.do(ctx, "POST", "/auth/users/register", req, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Get retrieves a user by ID
func (c *UsersClient) Get(ctx context.Context, id uint) (*User, error) {
	var user User
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/users/%d", id), nil, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email
func (c *UsersClient) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/users/email/%s", email), nil, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates a user
func (c *UsersClient) Update(ctx context.Context, id uint, req *UpdateUserRequest) (*User, error) {
	var user User
	err := c.do(ctx, "PUT", fmt.Sprintf("/auth/users/%d", id), req, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Delete deletes a user
func (c *UsersClient) Delete(ctx context.Context, id uint) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/auth/users/%d", id), nil, nil)
}

// List retrieves all users
func (c *UsersClient) List(ctx context.Context) ([]User, error) {
	var users []User
	err := c.do(ctx, "GET", "/auth/users", nil, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// GetPending retrieves all pending users
func (c *UsersClient) GetPending(ctx context.Context) ([]User, error) {
	var users []User
	err := c.do(ctx, "GET", "/auth/pending/users", nil, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// Activate activates a user
func (c *UsersClient) Activate(ctx context.Context, id uint) (*User, error) {
	var user User
	err := c.do(ctx, "POST", fmt.Sprintf("/auth/users/%d/activate", id), nil, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Deactivate deactivates a user
func (c *UsersClient) Deactivate(ctx context.Context, id uint) (*User, error) {
	var user User
	err := c.do(ctx, "POST", fmt.Sprintf("/auth/users/%d/deactivate", id), nil, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

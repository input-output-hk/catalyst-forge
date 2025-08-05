package users

import (
	"context"
	"fmt"
)

// RolesClientInterface defines the interface for role operations
type RolesClientInterface interface {
	Create(ctx context.Context, req *CreateRoleRequest) (*Role, error)
	CreateWithAdmin(ctx context.Context, req *CreateRoleRequest) (*Role, error)
	Get(ctx context.Context, id uint) (*Role, error)
	GetByName(ctx context.Context, name string) (*Role, error)
	Update(ctx context.Context, id uint, req *UpdateRoleRequest) (*Role, error)
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context) ([]Role, error)
	AssignUser(ctx context.Context, userID uint, roleID uint) error
	RemoveUser(ctx context.Context, userID uint, roleID uint) error
	GetUserRoles(ctx context.Context, userID uint) ([]UserRole, error)
	GetRoleUsers(ctx context.Context, roleID uint) ([]UserRole, error)
}

// RolesClient handles role-related operations
type RolesClient struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

// Ensure RolesClient implements RolesClientInterface
var _ RolesClientInterface = (*RolesClient)(nil)

// NewRolesClient creates a new roles client
func NewRolesClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *RolesClient {
	return &RolesClient{do: do}
}

// Create creates a new role
func (c *RolesClient) Create(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	var role Role
	err := c.do(ctx, "POST", "/auth/roles", req, &role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// CreateWithAdmin creates a role with admin privileges (all permissions)
func (c *RolesClient) CreateWithAdmin(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	var role Role
	err := c.do(ctx, "POST", "/auth/roles?admin=true", req, &role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// Get retrieves a role by ID
func (c *RolesClient) Get(ctx context.Context, id uint) (*Role, error) {
	var role Role
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/roles/%d", id), nil, &role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByName retrieves a role by name
func (c *RolesClient) GetByName(ctx context.Context, name string) (*Role, error) {
	var role Role
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/roles/name/%s", name), nil, &role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// Update updates a role
func (c *RolesClient) Update(ctx context.Context, id uint, req *UpdateRoleRequest) (*Role, error) {
	var role Role
	err := c.do(ctx, "PUT", fmt.Sprintf("/auth/roles/%d", id), req, &role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// Delete deletes a role
func (c *RolesClient) Delete(ctx context.Context, id uint) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/auth/roles/%d", id), nil, nil)
}

// List retrieves all roles
func (c *RolesClient) List(ctx context.Context) ([]Role, error) {
	var roles []Role
	err := c.do(ctx, "GET", "/auth/roles", nil, &roles)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// AssignUser assigns a user to a role
func (c *RolesClient) AssignUser(ctx context.Context, userID uint, roleID uint) error {
	return c.do(ctx, "POST", fmt.Sprintf("/auth/user-roles?user_id=%d&role_id=%d", userID, roleID), nil, nil)
}

// RemoveUser removes a user from a role
func (c *RolesClient) RemoveUser(ctx context.Context, userID uint, roleID uint) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/auth/user-roles?user_id=%d&role_id=%d", userID, roleID), nil, nil)
}

// GetUserRoles retrieves all roles for a user
func (c *RolesClient) GetUserRoles(ctx context.Context, userID uint) ([]UserRole, error) {
	var userRoles []UserRole
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/user-roles?user_id=%d", userID), nil, &userRoles)
	if err != nil {
		return nil, err
	}
	return userRoles, nil
}

// GetRoleUsers retrieves all users for a role
func (c *RolesClient) GetRoleUsers(ctx context.Context, roleID uint) ([]UserRole, error) {
	var userRoles []UserRole
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/role-users?role_id=%d", roleID), nil, &userRoles)
	if err != nil {
		return nil, err
	}
	return userRoles, nil
}

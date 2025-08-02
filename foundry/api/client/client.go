package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/client.go . Client

var ErrStatusConflict = errors.New("Resource already exists")

// Client interface defines the operations that can be performed against the API
type Client interface {
	// GHA operations
	ValidateToken(ctx context.Context, req *ValidateTokenRequest) (*ValidateTokenResponse, error)
	CreateAuth(ctx context.Context, req *CreateAuthRequest) (*GHARepositoryAuth, error)
	GetAuth(ctx context.Context, id uint) (*GHARepositoryAuth, error)
	GetAuthByRepository(ctx context.Context, repository string) (*GHARepositoryAuth, error)
	UpdateAuth(ctx context.Context, id uint, req *UpdateAuthRequest) (*GHARepositoryAuth, error)
	DeleteAuth(ctx context.Context, id uint) error
	ListAuths(ctx context.Context) ([]GHARepositoryAuth, error)

	// User operations
	CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
	RegisterUser(ctx context.Context, req *RegisterUserRequest) (*User, error)
	GetUser(ctx context.Context, id uint) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, id uint, req *UpdateUserRequest) (*User, error)
	DeleteUser(ctx context.Context, id uint) error
	ListUsers(ctx context.Context) ([]User, error)
	GetPendingUsers(ctx context.Context) ([]User, error)
	ActivateUser(ctx context.Context, id uint) (*User, error)
	DeactivateUser(ctx context.Context, id uint) (*User, error)

	// Role operations
	CreateRole(ctx context.Context, req *CreateRoleRequest) (*Role, error)
	CreateRoleWithAdmin(ctx context.Context, req *CreateRoleRequest) (*Role, error)
	GetRole(ctx context.Context, id uint) (*Role, error)
	GetRoleByName(ctx context.Context, name string) (*Role, error)
	UpdateRole(ctx context.Context, id uint, req *UpdateRoleRequest) (*Role, error)
	DeleteRole(ctx context.Context, id uint) error
	ListRoles(ctx context.Context) ([]Role, error)

	// User-role operations
	AssignUserToRole(ctx context.Context, userID uint, roleID uint) error
	RemoveUserFromRole(ctx context.Context, userID uint, roleID uint) error
	GetUserRoles(ctx context.Context, userID uint) ([]UserRole, error)
	GetRoleUsers(ctx context.Context, roleID uint) ([]UserRole, error)

	// User key operations
	CreateUserKey(ctx context.Context, req *CreateUserKeyRequest) (*UserKey, error)
	RegisterUserKey(ctx context.Context, req *RegisterUserKeyRequest) (*UserKey, error)
	GetUserKey(ctx context.Context, id uint) (*UserKey, error)
	GetUserKeyByKid(ctx context.Context, kid string) (*UserKey, error)
	UpdateUserKey(ctx context.Context, id uint, req *UpdateUserKeyRequest) (*UserKey, error)
	DeleteUserKey(ctx context.Context, id uint) error
	ListUserKeys(ctx context.Context) ([]UserKey, error)
	RevokeUserKey(ctx context.Context, id uint) (*UserKey, error)
	GetUserKeysByUserID(ctx context.Context, userID uint) ([]UserKey, error)
	GetActiveUserKeysByUserID(ctx context.Context, userID uint) ([]UserKey, error)
	GetInactiveUserKeysByUserID(ctx context.Context, userID uint) ([]UserKey, error)
	GetInactiveUserKeys(ctx context.Context) ([]UserKey, error)

	// Authentication operations
	CreateChallenge(ctx context.Context, req *ChallengeRequest) (*auth.KeyPairChallenge, error)
	Login(ctx context.Context, req *auth.KeyPairChallengeResponse) (*LoginResponse, error)

	// Release operations
	CreateRelease(ctx context.Context, release *Release, deploy bool) (*Release, error)
	GetRelease(ctx context.Context, id string) (*Release, error)
	UpdateRelease(ctx context.Context, release *Release) (*Release, error)
	ListReleases(ctx context.Context, projectName string) ([]Release, error)
	GetReleaseByAlias(ctx context.Context, aliasName string) (*Release, error)

	// Release alias operations
	CreateAlias(ctx context.Context, aliasName string, releaseID string) error
	DeleteAlias(ctx context.Context, aliasName string) error
	ListAliases(ctx context.Context, releaseID string) ([]ReleaseAlias, error)

	// Deployment operations
	CreateDeployment(ctx context.Context, releaseID string) (*ReleaseDeployment, error)
	GetDeployment(ctx context.Context, releaseID string, deployID string) (*ReleaseDeployment, error)
	UpdateDeployment(ctx context.Context, releaseID string, deployment *ReleaseDeployment) (*ReleaseDeployment, error)
	IncrementDeploymentAttempts(ctx context.Context, releaseID string, deployID string) (*ReleaseDeployment, error)
	ListDeployments(ctx context.Context, releaseID string) ([]ReleaseDeployment, error)
	GetLatestDeployment(ctx context.Context, releaseID string) (*ReleaseDeployment, error)

	// Deployment event operations
	AddDeploymentEvent(ctx context.Context, releaseID string, deployID string, name string, message string) (*ReleaseDeployment, error)
	GetDeploymentEvents(ctx context.Context, releaseID string, deployID string) ([]DeploymentEvent, error)
}

// HTTPClient is an implementation of the Client interface that uses HTTP
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// ClientOption is a function type for client configuration
type ClientOption func(*HTTPClient)

// WithTimeout sets the timeout for the HTTP client
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient.Timeout = timeout
	}
}

// WithTransport sets a custom transport for the HTTP client
func WithTransport(transport http.RoundTripper) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient.Transport = transport
	}
}

// WithToken sets the JWT token for authentication
func WithToken(token string) ClientOption {
	return func(c *HTTPClient) {
		c.token = token
	}
}

// NewClient creates a new API client
func NewClient(baseURL string, options ...ClientOption) Client {
	client := &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// do performs an HTTP request and processes the response
func (c *HTTPClient) do(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	var reqBodyReader io.Reader
	if reqBody != nil {
		reqBodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBodyReader = bytes.NewReader(reqBodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBodyReader)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Add JWT token to Authorization header if present
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error performing request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(respBodyBytes, &errResp); err != nil {
			return fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(respBodyBytes))
		}

		// Handle specific error cases
		if resp.StatusCode == http.StatusConflict {
			return ErrStatusConflict
		}

		return fmt.Errorf("API error: %d - %s", resp.StatusCode, errResp.Error)
	}

	if respBody != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.Unmarshal(respBodyBytes, respBody); err != nil {
			return fmt.Errorf("error unmarshaling response: %w", err)
		}
	}

	return nil
}

// User operations
func (c *HTTPClient) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
	var user User
	err := c.do(ctx, "POST", "/auth/users", req, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *HTTPClient) RegisterUser(ctx context.Context, req *RegisterUserRequest) (*User, error) {
	var user User
	err := c.do(ctx, "POST", "/auth/users/register", req, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *HTTPClient) GetUser(ctx context.Context, id uint) (*User, error) {
	var user User
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/users/%d", id), nil, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *HTTPClient) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/users/email/%s", email), nil, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *HTTPClient) UpdateUser(ctx context.Context, id uint, req *UpdateUserRequest) (*User, error) {
	var user User
	err := c.do(ctx, "PUT", fmt.Sprintf("/auth/users/%d", id), req, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *HTTPClient) DeleteUser(ctx context.Context, id uint) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/auth/users/%d", id), nil, nil)
}

func (c *HTTPClient) ListUsers(ctx context.Context) ([]User, error) {
	var users []User
	err := c.do(ctx, "GET", "/auth/users", nil, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (c *HTTPClient) GetPendingUsers(ctx context.Context) ([]User, error) {
	var users []User
	err := c.do(ctx, "GET", "/auth/pending/users", nil, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (c *HTTPClient) ActivateUser(ctx context.Context, id uint) (*User, error) {
	var user User
	err := c.do(ctx, "POST", fmt.Sprintf("/auth/users/%d/activate", id), nil, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *HTTPClient) DeactivateUser(ctx context.Context, id uint) (*User, error) {
	var user User
	err := c.do(ctx, "POST", fmt.Sprintf("/auth/users/%d/deactivate", id), nil, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Role operations
func (c *HTTPClient) CreateRole(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	var role Role
	err := c.do(ctx, "POST", "/auth/roles", req, &role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// CreateRoleWithAdmin creates a role with admin privileges (all permissions)
func (c *HTTPClient) CreateRoleWithAdmin(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	var role Role
	err := c.do(ctx, "POST", "/auth/roles?admin=true", req, &role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (c *HTTPClient) GetRole(ctx context.Context, id uint) (*Role, error) {
	var role Role
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/roles/%d", id), nil, &role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (c *HTTPClient) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	var role Role
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/roles/name/%s", name), nil, &role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (c *HTTPClient) UpdateRole(ctx context.Context, id uint, req *UpdateRoleRequest) (*Role, error) {
	var role Role
	err := c.do(ctx, "PUT", fmt.Sprintf("/auth/roles/%d", id), req, &role)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (c *HTTPClient) DeleteRole(ctx context.Context, id uint) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/auth/roles/%d", id), nil, nil)
}

func (c *HTTPClient) ListRoles(ctx context.Context) ([]Role, error) {
	var roles []Role
	err := c.do(ctx, "GET", "/auth/roles", nil, &roles)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// User-role operations
func (c *HTTPClient) AssignUserToRole(ctx context.Context, userID uint, roleID uint) error {
	return c.do(ctx, "POST", fmt.Sprintf("/auth/user-roles?user_id=%d&role_id=%d", userID, roleID), nil, nil)
}

func (c *HTTPClient) RemoveUserFromRole(ctx context.Context, userID uint, roleID uint) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/auth/user-roles?user_id=%d&role_id=%d", userID, roleID), nil, nil)
}

func (c *HTTPClient) GetUserRoles(ctx context.Context, userID uint) ([]UserRole, error) {
	var userRoles []UserRole
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/user-roles?user_id=%d", userID), nil, &userRoles)
	if err != nil {
		return nil, err
	}
	return userRoles, nil
}

func (c *HTTPClient) GetRoleUsers(ctx context.Context, roleID uint) ([]UserRole, error) {
	var userRoles []UserRole
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/role-users?role_id=%d", roleID), nil, &userRoles)
	if err != nil {
		return nil, err
	}
	return userRoles, nil
}

// User key operations
func (c *HTTPClient) CreateUserKey(ctx context.Context, req *CreateUserKeyRequest) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "POST", "/auth/keys", req, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

func (c *HTTPClient) RegisterUserKey(ctx context.Context, req *RegisterUserKeyRequest) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "POST", "/auth/keys/register", req, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

func (c *HTTPClient) GetUserKey(ctx context.Context, id uint) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/keys/%d", id), nil, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

func (c *HTTPClient) GetUserKeyByKid(ctx context.Context, kid string) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/keys/kid/%s", kid), nil, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

func (c *HTTPClient) UpdateUserKey(ctx context.Context, id uint, req *UpdateUserKeyRequest) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "PUT", fmt.Sprintf("/auth/keys/%d", id), req, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

func (c *HTTPClient) DeleteUserKey(ctx context.Context, id uint) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/auth/keys/%d", id), nil, nil)
}

func (c *HTTPClient) ListUserKeys(ctx context.Context) ([]UserKey, error) {
	var userKeys []UserKey
	err := c.do(ctx, "GET", "/auth/keys", nil, &userKeys)
	if err != nil {
		return nil, err
	}
	return userKeys, nil
}

func (c *HTTPClient) RevokeUserKey(ctx context.Context, id uint) (*UserKey, error) {
	var userKey UserKey
	err := c.do(ctx, "POST", fmt.Sprintf("/auth/keys/%d/revoke", id), nil, &userKey)
	if err != nil {
		return nil, err
	}
	return &userKey, nil
}

func (c *HTTPClient) GetUserKeysByUserID(ctx context.Context, userID uint) ([]UserKey, error) {
	var userKeys []UserKey
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/keys/user/%d", userID), nil, &userKeys)
	if err != nil {
		return nil, err
	}
	return userKeys, nil
}

func (c *HTTPClient) GetActiveUserKeysByUserID(ctx context.Context, userID uint) ([]UserKey, error) {
	var userKeys []UserKey
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/keys/user/%d/active", userID), nil, &userKeys)
	if err != nil {
		return nil, err
	}
	return userKeys, nil
}

func (c *HTTPClient) GetInactiveUserKeysByUserID(ctx context.Context, userID uint) ([]UserKey, error) {
	var userKeys []UserKey
	err := c.do(ctx, "GET", fmt.Sprintf("/auth/keys/user/%d/inactive", userID), nil, &userKeys)
	if err != nil {
		return nil, err
	}
	return userKeys, nil
}

func (c *HTTPClient) GetInactiveUserKeys(ctx context.Context) ([]UserKey, error) {
	var userKeys []UserKey
	err := c.do(ctx, "GET", "/auth/pending/keys", nil, &userKeys)
	if err != nil {
		return nil, err
	}
	return userKeys, nil
}

// Authentication operations
func (c *HTTPClient) CreateChallenge(ctx context.Context, req *ChallengeRequest) (*auth.KeyPairChallenge, error) {
	var challenge auth.KeyPairChallenge
	err := c.do(ctx, "POST", "/auth/challenge", req, &challenge)
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (c *HTTPClient) Login(ctx context.Context, req *auth.KeyPairChallengeResponse) (*LoginResponse, error) {
	var response LoginResponse
	err := c.do(ctx, "POST", "/auth/login", req, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

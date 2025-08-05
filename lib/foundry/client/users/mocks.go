package users

import (
	"context"
	"sync"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/users.go . UsersClientInterface
//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/roles.go . RolesClientInterface
//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/keys.go . KeysClientInterface

// MockUsersClient is a mock implementation of UsersClient for testing
type MockUsersClient struct {
	CreateFunc     func(ctx context.Context, req *CreateUserRequest) (*User, error)
	RegisterFunc   func(ctx context.Context, req *RegisterUserRequest) (*User, error)
	GetFunc        func(ctx context.Context, id uint) (*User, error)
	GetByEmailFunc func(ctx context.Context, email string) (*User, error)
	UpdateFunc     func(ctx context.Context, id uint, req *UpdateUserRequest) (*User, error)
	DeleteFunc     func(ctx context.Context, id uint) error
	ListFunc       func(ctx context.Context) ([]User, error)
	GetPendingFunc func(ctx context.Context) ([]User, error)
	ActivateFunc   func(ctx context.Context, id uint) (*User, error)
	DeactivateFunc func(ctx context.Context, id uint) (*User, error)

	mu sync.RWMutex
}

func (m *MockUsersClient) Create(ctx context.Context, req *CreateUserRequest) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockUsersClient) Register(ctx context.Context, req *RegisterUserRequest) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockUsersClient) Get(ctx context.Context, id uint) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockUsersClient) GetByEmail(ctx context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockUsersClient) Update(ctx context.Context, id uint, req *UpdateUserRequest) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return nil, nil
}

func (m *MockUsersClient) Delete(ctx context.Context, id uint) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockUsersClient) List(ctx context.Context) ([]User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

func (m *MockUsersClient) GetPending(ctx context.Context) ([]User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetPendingFunc != nil {
		return m.GetPendingFunc(ctx)
	}
	return nil, nil
}

func (m *MockUsersClient) Activate(ctx context.Context, id uint) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ActivateFunc != nil {
		return m.ActivateFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockUsersClient) Deactivate(ctx context.Context, id uint) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.DeactivateFunc != nil {
		return m.DeactivateFunc(ctx, id)
	}
	return nil, nil
}

// MockRolesClient is a mock implementation of RolesClient for testing
type MockRolesClient struct {
	CreateFunc          func(ctx context.Context, req *CreateRoleRequest) (*Role, error)
	CreateWithAdminFunc func(ctx context.Context, req *CreateRoleRequest) (*Role, error)
	GetFunc             func(ctx context.Context, id uint) (*Role, error)
	GetByNameFunc       func(ctx context.Context, name string) (*Role, error)
	UpdateFunc          func(ctx context.Context, id uint, req *UpdateRoleRequest) (*Role, error)
	DeleteFunc          func(ctx context.Context, id uint) error
	ListFunc            func(ctx context.Context) ([]Role, error)
	AssignUserFunc      func(ctx context.Context, userID uint, roleID uint) error
	RemoveUserFunc      func(ctx context.Context, userID uint, roleID uint) error
	GetUserRolesFunc    func(ctx context.Context, userID uint) ([]UserRole, error)
	GetRoleUsersFunc    func(ctx context.Context, roleID uint) ([]UserRole, error)

	mu sync.RWMutex
}

func (m *MockRolesClient) Create(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockRolesClient) CreateWithAdmin(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.CreateWithAdminFunc != nil {
		return m.CreateWithAdminFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockRolesClient) Get(ctx context.Context, id uint) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockRolesClient) GetByName(ctx context.Context, name string) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetByNameFunc != nil {
		return m.GetByNameFunc(ctx, name)
	}
	return nil, nil
}

func (m *MockRolesClient) Update(ctx context.Context, id uint, req *UpdateRoleRequest) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return nil, nil
}

func (m *MockRolesClient) Delete(ctx context.Context, id uint) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockRolesClient) List(ctx context.Context) ([]Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

func (m *MockRolesClient) AssignUser(ctx context.Context, userID uint, roleID uint) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.AssignUserFunc != nil {
		return m.AssignUserFunc(ctx, userID, roleID)
	}
	return nil
}

func (m *MockRolesClient) RemoveUser(ctx context.Context, userID uint, roleID uint) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.RemoveUserFunc != nil {
		return m.RemoveUserFunc(ctx, userID, roleID)
	}
	return nil
}

func (m *MockRolesClient) GetUserRoles(ctx context.Context, userID uint) ([]UserRole, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetUserRolesFunc != nil {
		return m.GetUserRolesFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockRolesClient) GetRoleUsers(ctx context.Context, roleID uint) ([]UserRole, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetRoleUsersFunc != nil {
		return m.GetRoleUsersFunc(ctx, roleID)
	}
	return nil, nil
}

// MockKeysClient is a mock implementation of KeysClient for testing
type MockKeysClient struct {
	CreateFunc              func(ctx context.Context, req *CreateUserKeyRequest) (*UserKey, error)
	RegisterFunc            func(ctx context.Context, req *RegisterUserKeyRequest) (*UserKey, error)
	GetFunc                 func(ctx context.Context, id uint) (*UserKey, error)
	GetByKidFunc            func(ctx context.Context, kid string) (*UserKey, error)
	UpdateFunc              func(ctx context.Context, id uint, req *UpdateUserKeyRequest) (*UserKey, error)
	DeleteFunc              func(ctx context.Context, id uint) error
	ListFunc                func(ctx context.Context) ([]UserKey, error)
	RevokeFunc              func(ctx context.Context, id uint) (*UserKey, error)
	GetByUserIDFunc         func(ctx context.Context, userID uint) ([]UserKey, error)
	GetActiveByUserIDFunc   func(ctx context.Context, userID uint) ([]UserKey, error)
	GetInactiveByUserIDFunc func(ctx context.Context, userID uint) ([]UserKey, error)
	GetInactiveFunc         func(ctx context.Context) ([]UserKey, error)

	mu sync.RWMutex
}

func (m *MockKeysClient) Create(ctx context.Context, req *CreateUserKeyRequest) (*UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockKeysClient) Register(ctx context.Context, req *RegisterUserKeyRequest) (*UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockKeysClient) Get(ctx context.Context, id uint) (*UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockKeysClient) GetByKid(ctx context.Context, kid string) (*UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetByKidFunc != nil {
		return m.GetByKidFunc(ctx, kid)
	}
	return nil, nil
}

func (m *MockKeysClient) Update(ctx context.Context, id uint, req *UpdateUserKeyRequest) (*UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return nil, nil
}

func (m *MockKeysClient) Delete(ctx context.Context, id uint) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockKeysClient) List(ctx context.Context) ([]UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return nil, nil
}

func (m *MockKeysClient) Revoke(ctx context.Context, id uint) (*UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.RevokeFunc != nil {
		return m.RevokeFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockKeysClient) GetByUserID(ctx context.Context, userID uint) ([]UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockKeysClient) GetActiveByUserID(ctx context.Context, userID uint) ([]UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetActiveByUserIDFunc != nil {
		return m.GetActiveByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockKeysClient) GetInactiveByUserID(ctx context.Context, userID uint) ([]UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetInactiveByUserIDFunc != nil {
		return m.GetInactiveByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockKeysClient) GetInactive(ctx context.Context) ([]UserKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.GetInactiveFunc != nil {
		return m.GetInactiveFunc(ctx)
	}
	return nil, nil
}

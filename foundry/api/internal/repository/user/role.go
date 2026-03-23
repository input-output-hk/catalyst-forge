package user

import (
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

// RoleRepository defines the interface for role repository operations
type RoleRepository interface {
	Create(role *user.Role) error
	GetByID(id string) (*user.Role, error)
	GetByName(name string) (*user.Role, error)
	Update(role *user.Role) error
	Delete(id string) error
	List() ([]user.Role, error)
}

// DefaultRoleRepository is the default implementation of RoleRepository
type DefaultRoleRepository struct {
	db *gorm.DB
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *gorm.DB) *DefaultRoleRepository {
	return &DefaultRoleRepository{
		db: db,
	}
}

// Create creates a new role
func (r *DefaultRoleRepository) Create(role *user.Role) error {
	return r.db.Create(role).Error
}

// GetByID retrieves a role by ID
func (r *DefaultRoleRepository) GetByID(id string) (*user.Role, error) {
	var role user.Role
	if err := r.db.First(&role, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByName retrieves a role by name
func (r *DefaultRoleRepository) GetByName(name string) (*user.Role, error) {
	var role user.Role
	if err := r.db.Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// Update updates an existing role
func (r *DefaultRoleRepository) Update(role *user.Role) error {
	return r.db.Save(role).Error
}

// Delete deletes a role
func (r *DefaultRoleRepository) Delete(id string) error {
	return r.db.Delete(&user.Role{}, "id = ?", id).Error
}

// List retrieves all roles
func (r *DefaultRoleRepository) List() ([]user.Role, error) {
	var roles []user.Role
	if err := r.db.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

package user

import (
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

// UserRoleRepository defines the interface for user role repository operations
type UserRoleRepository interface {
	Create(userRole *user.UserRole) error
	GetByID(id string) (*user.UserRole, error)
	GetByUserID(userID string) ([]user.UserRole, error)
	GetByRoleID(roleID string) ([]user.UserRole, error)
	DeleteByUserIDAndRoleID(userID, roleID string) error
	DeleteByUserID(userID string) error
	List() ([]user.UserRole, error)
}

// DefaultUserRoleRepository is the default implementation of UserRoleRepository
type DefaultUserRoleRepository struct {
	db *gorm.DB
}

// NewUserRoleRepository creates a new user role repository
func NewUserRoleRepository(db *gorm.DB) *DefaultUserRoleRepository {
	return &DefaultUserRoleRepository{
		db: db,
	}
}

// Create creates a new user role relationship
func (r *DefaultUserRoleRepository) Create(userRole *user.UserRole) error {
	return r.db.Create(userRole).Error
}

// GetByID retrieves a user role relationship by ID
func (r *DefaultUserRoleRepository) GetByID(id string) (*user.UserRole, error) {
	var userRole user.UserRole
	if err := r.db.First(&userRole, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &userRole, nil
}

// GetByUserID retrieves all roles for a specific user
func (r *DefaultUserRoleRepository) GetByUserID(userID string) ([]user.UserRole, error) {
	var userRoles []user.UserRole
	if err := r.db.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}
	return userRoles, nil
}

// GetByRoleID retrieves all users for a specific role
func (r *DefaultUserRoleRepository) GetByRoleID(roleID string) ([]user.UserRole, error) {
	var userRoles []user.UserRole
	if err := r.db.Where("role_id = ?", roleID).Find(&userRoles).Error; err != nil {
		return nil, err
	}
	return userRoles, nil
}

// DeleteByUserIDAndRoleID deletes a specific user role relationship
func (r *DefaultUserRoleRepository) DeleteByUserIDAndRoleID(userID, roleID string) error {
	return r.db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&user.UserRole{}).Error
}

// DeleteByUserID deletes all roles for a specific user
func (r *DefaultUserRoleRepository) DeleteByUserID(userID string) error {
	return r.db.Where("user_id = ?", userID).Delete(&user.UserRole{}).Error
}

// List retrieves all user role relationships
func (r *DefaultUserRoleRepository) List() ([]user.UserRole, error) {
	var userRoles []user.UserRole
	if err := r.db.Find(&userRoles).Error; err != nil {
		return nil, err
	}
	return userRoles, nil
}

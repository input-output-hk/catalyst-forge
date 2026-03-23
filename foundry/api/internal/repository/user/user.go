package user

import (
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

// UserRepository defines the interface for user repository operations
type UserRepository interface {
	Create(user *user.User) error
	GetByID(id uint) (*user.User, error)
	GetByEmail(email string) (*user.User, error)
	Update(user *user.User) error
	Delete(id uint) error
	List() ([]user.User, error)
	GetByStatus(status user.UserStatus) ([]user.User, error)
}

// DefaultUserRepository is the default implementation of UserRepository
type DefaultUserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *DefaultUserRepository {
	return &DefaultUserRepository{
		db: db,
	}
}

// Create creates a new user
func (r *DefaultUserRepository) Create(user *user.User) error {
	return r.db.Create(user).Error
}

// GetByID retrieves a user by ID
func (r *DefaultUserRepository) GetByID(id uint) (*user.User, error) {
	var u user.User
	if err := r.db.First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByEmail retrieves a user by email
func (r *DefaultUserRepository) GetByEmail(email string) (*user.User, error) {
	var u user.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Update updates an existing user
func (r *DefaultUserRepository) Update(user *user.User) error {
	return r.db.Save(user).Error
}

// Delete deletes a user
func (r *DefaultUserRepository) Delete(id uint) error {
	return r.db.Delete(&user.User{}, "id = ?", id).Error
}

// List retrieves all users
func (r *DefaultUserRepository) List() ([]user.User, error) {
	var users []user.User
	if err := r.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetByStatus retrieves all users with a specific status
func (r *DefaultUserRepository) GetByStatus(status user.UserStatus) ([]user.User, error) {
	var users []user.User
	if err := r.db.Where("status = ?", status).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

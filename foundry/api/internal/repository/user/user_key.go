package user

import (
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

// UserKeyRepository defines the interface for user key repository operations
type UserKeyRepository interface {
	Create(userKey *user.UserKey) error
	GetByID(id uint) (*user.UserKey, error)
	GetByKid(kid string) (*user.UserKey, error)
	GetByUserID(userID uint) ([]user.UserKey, error)
	GetActiveByUserID(userID uint) ([]user.UserKey, error)
	GetInactiveByUserID(userID uint) ([]user.UserKey, error)
	GetInactive() ([]user.UserKey, error)
	Update(userKey *user.UserKey) error
	Delete(id uint) error
	List() ([]user.UserKey, error)
}

// DefaultUserKeyRepository is the default implementation of UserKeyRepository
type DefaultUserKeyRepository struct {
	db *gorm.DB
}

// NewUserKeyRepository creates a new user key repository
func NewUserKeyRepository(db *gorm.DB) *DefaultUserKeyRepository {
	return &DefaultUserKeyRepository{
		db: db,
	}
}

// Create creates a new user key
func (r *DefaultUserKeyRepository) Create(userKey *user.UserKey) error {
	return r.db.Create(userKey).Error
}

// GetByID retrieves a user key by ID
func (r *DefaultUserKeyRepository) GetByID(id uint) (*user.UserKey, error) {
	var userKey user.UserKey
	if err := r.db.First(&userKey, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &userKey, nil
}

// GetByKid retrieves a user key by kid (key ID)
func (r *DefaultUserKeyRepository) GetByKid(kid string) (*user.UserKey, error) {
	var userKey user.UserKey
	if err := r.db.Where("kid = ?", kid).First(&userKey).Error; err != nil {
		return nil, err
	}
	return &userKey, nil
}

// GetByUserID retrieves all keys for a specific user
func (r *DefaultUserKeyRepository) GetByUserID(userID uint) ([]user.UserKey, error) {
	var userKeys []user.UserKey
	if err := r.db.Where("user_id = ?", userID).Find(&userKeys).Error; err != nil {
		return nil, err
	}
	return userKeys, nil
}

// GetActiveByUserID retrieves all active keys for a specific user
func (r *DefaultUserKeyRepository) GetActiveByUserID(userID uint) ([]user.UserKey, error) {
	var userKeys []user.UserKey
	if err := r.db.Where("user_id = ? AND status = ?", userID, user.UserKeyStatusActive).Find(&userKeys).Error; err != nil {
		return nil, err
	}
	return userKeys, nil
}

// GetInactiveByUserID retrieves all inactive keys for a specific user
func (r *DefaultUserKeyRepository) GetInactiveByUserID(userID uint) ([]user.UserKey, error) {
	var userKeys []user.UserKey
	if err := r.db.Where("user_id = ? AND status = ?", userID, user.UserKeyStatusInactive).Find(&userKeys).Error; err != nil {
		return nil, err
	}
	return userKeys, nil
}

// GetInactive retrieves all inactive keys
func (r *DefaultUserKeyRepository) GetInactive() ([]user.UserKey, error) {
	var userKeys []user.UserKey
	if err := r.db.Where("status = ?", user.UserKeyStatusInactive).Find(&userKeys).Error; err != nil {
		return nil, err
	}
	return userKeys, nil
}

// Update updates an existing user key
func (r *DefaultUserKeyRepository) Update(userKey *user.UserKey) error {
	return r.db.Save(userKey).Error
}

// Delete deletes a user key
func (r *DefaultUserKeyRepository) Delete(id uint) error {
	return r.db.Delete(&user.UserKey{}, "id = ?", id).Error
}

// List retrieves all user keys
func (r *DefaultUserKeyRepository) List() ([]user.UserKey, error) {
	var userKeys []user.UserKey
	if err := r.db.Find(&userKeys).Error; err != nil {
		return nil, err
	}
	return userKeys, nil
}

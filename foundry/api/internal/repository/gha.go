package repository

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
	"gorm.io/gorm"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/gha_auth.go . GHAAuthRepository

// GHAAuthRepository defines the interface for GitHub Actions authentication repository operations
type GHAAuthRepository interface {
	Create(auth *models.GHARepositoryAuth) error
	GetByID(id uint) (*models.GHARepositoryAuth, error)
	GetByRepository(repository string) (*models.GHARepositoryAuth, error)
	Update(auth *models.GHARepositoryAuth) error
	Delete(id uint) error
	List() ([]models.GHARepositoryAuth, error)
	GetPermissionsForRepository(repository string) ([]auth.Permission, error)
}

// DefaultGHAAuthRepository is the default implementation of GHAAuthRepository
type DefaultGHAAuthRepository struct {
	db *gorm.DB
}

// NewGHAAuthRepository creates a new GitHub Actions authentication repository
func NewGHAAuthRepository(db *gorm.DB) *DefaultGHAAuthRepository {
	return &DefaultGHAAuthRepository{
		db: db,
	}
}

// Create creates a new GitHub Actions authentication configuration
func (r *DefaultGHAAuthRepository) Create(auth *models.GHARepositoryAuth) error {
	return r.db.Create(auth).Error
}

// GetByID retrieves a GitHub Actions authentication configuration by ID
func (r *DefaultGHAAuthRepository) GetByID(id uint) (*models.GHARepositoryAuth, error) {
	var auth models.GHARepositoryAuth
	if err := r.db.First(&auth, id).Error; err != nil {
		return nil, err
	}
	return &auth, nil
}

// GetByRepository retrieves a GitHub Actions authentication configuration by repository name
func (r *DefaultGHAAuthRepository) GetByRepository(repository string) (*models.GHARepositoryAuth, error) {
	var auth models.GHARepositoryAuth
	if err := r.db.Where("repository = ?", repository).First(&auth).Error; err != nil {
		return nil, err
	}
	return &auth, nil
}

// Update updates an existing GitHub Actions authentication configuration
func (r *DefaultGHAAuthRepository) Update(auth *models.GHARepositoryAuth) error {
	return r.db.Save(auth).Error
}

// Delete deletes a GitHub Actions authentication configuration
func (r *DefaultGHAAuthRepository) Delete(id uint) error {
	return r.db.Delete(&models.GHARepositoryAuth{}, id).Error
}

// List retrieves all GitHub Actions authentication configurations
func (r *DefaultGHAAuthRepository) List() ([]models.GHARepositoryAuth, error) {
	var auths []models.GHARepositoryAuth
	if err := r.db.Find(&auths).Error; err != nil {
		return nil, err
	}
	return auths, nil
}

// GetPermissionsForRepository retrieves the permissions for a specific repository
func (r *DefaultGHAAuthRepository) GetPermissionsForRepository(repository string) ([]auth.Permission, error) {
	var auth models.GHARepositoryAuth
	if err := r.db.Where("repository = ? AND enabled = ?", repository, true).First(&auth).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no authentication configuration found for repository: %s", repository)
		}
		return nil, err
	}
	return auth.GetPermissions(), nil
}

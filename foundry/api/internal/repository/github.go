package repository

import (
	"fmt"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"gorm.io/gorm"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/gha_auth.go . GithubAuthRepository

// GithubAuthRepository defines the interface for GitHub Actions authentication repository operations
type GithubAuthRepository interface {
	Create(auth *models.GithubRepositoryAuth) error
	GetByID(id uint) (*models.GithubRepositoryAuth, error)
	GetByRepository(repository string) (*models.GithubRepositoryAuth, error)
	Update(auth *models.GithubRepositoryAuth) error
	Delete(id uint) error
	List() ([]models.GithubRepositoryAuth, error)
	GetPermissionsForRepository(repository string) ([]auth.Permission, error)
}

// DefaultGithubAuthRepository is the default implementation of GithubAuthRepository
type DefaultGithubAuthRepository struct {
	db *gorm.DB
}

// NewGithubAuthRepository creates a new GitHub Actions authentication repository
func NewGithubAuthRepository(db *gorm.DB) *DefaultGithubAuthRepository {
	return &DefaultGithubAuthRepository{
		db: db,
	}
}

// Create creates a new GitHub Actions authentication configuration
func (r *DefaultGithubAuthRepository) Create(auth *models.GithubRepositoryAuth) error {
	return r.db.Create(auth).Error
}

// GetByID retrieves a GitHub Actions authentication configuration by ID
func (r *DefaultGithubAuthRepository) GetByID(id uint) (*models.GithubRepositoryAuth, error) {
	var auth models.GithubRepositoryAuth
	if err := r.db.First(&auth, id).Error; err != nil {
		return nil, err
	}
	return &auth, nil
}

// GetByRepository retrieves a GitHub Actions authentication configuration by repository name
func (r *DefaultGithubAuthRepository) GetByRepository(repository string) (*models.GithubRepositoryAuth, error) {
	var auth models.GithubRepositoryAuth
	if err := r.db.Where("repository = ?", repository).First(&auth).Error; err != nil {
		return nil, err
	}
	return &auth, nil
}

// Update updates an existing GitHub Actions authentication configuration
func (r *DefaultGithubAuthRepository) Update(auth *models.GithubRepositoryAuth) error {
	return r.db.Save(auth).Error
}

// Delete deletes a GitHub Actions authentication configuration
func (r *DefaultGithubAuthRepository) Delete(id uint) error {
	return r.db.Delete(&models.GithubRepositoryAuth{}, id).Error
}

// List retrieves all GitHub Actions authentication configurations
func (r *DefaultGithubAuthRepository) List() ([]models.GithubRepositoryAuth, error) {
	var auths []models.GithubRepositoryAuth
	if err := r.db.Find(&auths).Error; err != nil {
		return nil, err
	}
	return auths, nil
}

// GetPermissionsForRepository retrieves the permissions for a specific repository
func (r *DefaultGithubAuthRepository) GetPermissionsForRepository(repository string) ([]auth.Permission, error) {
	var auth models.GithubRepositoryAuth
	if err := r.db.Where("repository = ? AND enabled = ?", repository, true).First(&auth).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no authentication configuration found for repository: %s", repository)
		}
		return nil, err
	}
	return auth.GetPermissions(), nil
}

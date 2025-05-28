package repository

import (
	"context"
	"errors"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"gorm.io/gorm"
)

// ReleaseRepository defines the interface for release operations
type ReleaseRepository interface {
	Create(ctx context.Context, release *models.Release) error
	GetByID(ctx context.Context, id string) (*models.Release, error)
	Update(ctx context.Context, release *models.Release) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, projectName string) ([]models.Release, error)
	ListAll(ctx context.Context) ([]models.Release, error)
	GetByAlias(ctx context.Context, aliasName string) (*models.Release, error)
}

// GormReleaseRepository implements ReleaseRepository using GORM
type GormReleaseRepository struct {
	db *gorm.DB
}

// NewReleaseRepository creates a new ReleaseRepository
func NewReleaseRepository(db *gorm.DB) ReleaseRepository {
	return &GormReleaseRepository{db: db}
}

// Create adds a new release to the database
func (r *GormReleaseRepository) Create(ctx context.Context, release *models.Release) error {
	return r.db.WithContext(ctx).Create(release).Error
}

// GetByID retrieves a release by its ID
func (r *GormReleaseRepository) GetByID(ctx context.Context, id string) (*models.Release, error) {
	var release models.Release
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&release).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("release not found")
		}
		return nil, err
	}
	return &release, nil
}

// Update modifies an existing release
func (r *GormReleaseRepository) Update(ctx context.Context, release *models.Release) error {
	return r.db.WithContext(ctx).Save(release).Error
}

// Delete removes a release (soft delete)
func (r *GormReleaseRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Release{}).Error
}

// List retrieves releases filtered by project
func (r *GormReleaseRepository) List(ctx context.Context, projectName string) ([]models.Release, error) {
	var releases []models.Release
	query := r.db.WithContext(ctx).Where("project = ?", projectName)

	if err := query.Order("created_at DESC").Find(&releases).Error; err != nil {
		return nil, err
	}

	return releases, nil
}

// ListAll retrieves all releases
func (r *GormReleaseRepository) ListAll(ctx context.Context) ([]models.Release, error) {
	var releases []models.Release

	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&releases).Error; err != nil {
		return nil, err
	}

	return releases, nil
}

// GetByAlias retrieves a release by its alias name
func (r *GormReleaseRepository) GetByAlias(ctx context.Context, aliasName string) (*models.Release, error) {
	var alias models.ReleaseAlias
	if err := r.db.WithContext(ctx).Where("name = ?", aliasName).First(&alias).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("alias not found")
		}
		return nil, err
	}

	return r.GetByID(ctx, alias.ReleaseID)
}

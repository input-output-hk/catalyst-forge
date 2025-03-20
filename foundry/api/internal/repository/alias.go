package repository

import (
	"context"
	"errors"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"gorm.io/gorm"
)

// AliasRepository defines the interface for release alias operations
type AliasRepository interface {
	Create(ctx context.Context, alias *models.ReleaseAlias) error
	Get(ctx context.Context, name string) (*models.ReleaseAlias, error)
	Update(ctx context.Context, alias *models.ReleaseAlias) error
	Delete(ctx context.Context, name string) error
	ListByReleaseID(ctx context.Context, releaseID string) ([]models.ReleaseAlias, error)
}

// GormAliasRepository implements AliasRepository using GORM
type GormAliasRepository struct {
	db *gorm.DB
}

// NewAliasRepository creates a new AliasRepository
func NewAliasRepository(db *gorm.DB) AliasRepository {
	return &GormAliasRepository{db: db}
}

// Create adds a new release alias to the database
func (r *GormAliasRepository) Create(ctx context.Context, alias *models.ReleaseAlias) error {
	return r.db.WithContext(ctx).Create(alias).Error
}

// Get retrieves an alias by its name
func (r *GormAliasRepository) Get(ctx context.Context, name string) (*models.ReleaseAlias, error) {
	var alias models.ReleaseAlias
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&alias).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("alias not found")
		}
		return nil, err
	}
	return &alias, nil
}

// Update modifies an existing alias
func (r *GormAliasRepository) Update(ctx context.Context, alias *models.ReleaseAlias) error {
	return r.db.WithContext(ctx).Save(alias).Error
}

// Delete removes an alias (soft delete)
func (r *GormAliasRepository) Delete(ctx context.Context, name string) error {
	return r.db.WithContext(ctx).Where("name = ?", name).Delete(&models.ReleaseAlias{}).Error
}

// ListByReleaseID retrieves all aliases for a specific release
func (r *GormAliasRepository) ListByReleaseID(ctx context.Context, releaseID string) ([]models.ReleaseAlias, error) {
	var aliases []models.ReleaseAlias
	if err := r.db.WithContext(ctx).Where("release_id = ?", releaseID).Find(&aliases).Error; err != nil {
		return nil, err
	}
	return aliases, nil
}

package release

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/release"
	base "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
)

var (
	ErrInjectionNotFound = errors.New("release injection not found")
	ErrInjectionExists   = errors.New("release injection already exists")
)

// InjectionRepository defines the interface for release injection operations
type InjectionRepository interface {
	Create(ctx context.Context, injection *release.ReleaseInjection) error
	CreateBulk(ctx context.Context, injections []release.ReleaseInjection) error
	GetByID(ctx context.Context, id uuid.UUID) (*release.ReleaseInjection, error)
	GetByReleaseAndPointer(ctx context.Context, releaseID uuid.UUID, jsonPointer string) (*release.ReleaseInjection, error)
	ListByRelease(ctx context.Context, releaseID uuid.UUID) ([]release.ReleaseInjection, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByRelease(ctx context.Context, releaseID uuid.UUID) error
}

// injectionRepositoryImpl implements InjectionRepository interface
type injectionRepositoryImpl struct {
	db *gorm.DB
}

// NewInjectionRepository creates a new injection repository instance
func NewInjectionRepository(db *gorm.DB) InjectionRepository {
	return &injectionRepositoryImpl{db: db}
}

// Create creates a new release injection
func (r *injectionRepositoryImpl) Create(ctx context.Context, injection *release.ReleaseInjection) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(injection).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrInjectionExists
		}
		return err
	}
	
	return nil
}

// CreateBulk creates multiple release injections
func (r *injectionRepositoryImpl) CreateBulk(ctx context.Context, injections []release.ReleaseInjection) error {
	if len(injections) == 0 {
		return nil
	}
	
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(&injections).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrInjectionExists
		}
		return err
	}
	
	return nil
}

// GetByID retrieves a release injection by ID
func (r *injectionRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*release.ReleaseInjection, error) {
	db := base.GetDB(ctx, r.db)
	
	var injection release.ReleaseInjection
	if err := db.Where("id = ?", id).First(&injection).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInjectionNotFound
		}
		return nil, err
	}
	
	return &injection, nil
}

// GetByReleaseAndPointer retrieves a release injection by release ID and JSON pointer
func (r *injectionRepositoryImpl) GetByReleaseAndPointer(ctx context.Context, releaseID uuid.UUID, jsonPointer string) (*release.ReleaseInjection, error) {
	db := base.GetDB(ctx, r.db)
	
	var injection release.ReleaseInjection
	if err := db.Where("release_id = ? AND json_pointer = ?", releaseID, jsonPointer).First(&injection).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInjectionNotFound
		}
		return nil, err
	}
	
	return &injection, nil
}

// ListByRelease retrieves all injections for a release
func (r *injectionRepositoryImpl) ListByRelease(ctx context.Context, releaseID uuid.UUID) ([]release.ReleaseInjection, error) {
	db := base.GetDB(ctx, r.db)
	
	var injections []release.ReleaseInjection
	if err := db.Where("release_id = ?", releaseID).Order("json_pointer").Find(&injections).Error; err != nil {
		return nil, err
	}
	
	return injections, nil
}

// Delete deletes a release injection by ID
func (r *injectionRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Where("id = ?", id).Delete(&release.ReleaseInjection{})
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrInjectionNotFound
	}
	
	return nil
}

// DeleteByRelease deletes all injections for a release
func (r *injectionRepositoryImpl) DeleteByRelease(ctx context.Context, releaseID uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Where("release_id = ?", releaseID).Delete(&release.ReleaseInjection{}).Error; err != nil {
		return err
	}
	
	return nil
}
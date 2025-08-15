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
	ErrModuleNotFound = errors.New("release module not found")
	ErrModuleExists   = errors.New("release module already exists")
)

// ModuleRepository defines the interface for release module operations
type ModuleRepository interface {
	Create(ctx context.Context, module *release.ReleaseModule) error
	CreateBulk(ctx context.Context, modules []release.ReleaseModule) error
	GetByID(ctx context.Context, id uuid.UUID) (*release.ReleaseModule, error)
	GetByReleaseAndKey(ctx context.Context, releaseID uuid.UUID, moduleKey string) (*release.ReleaseModule, error)
	ListByRelease(ctx context.Context, releaseID uuid.UUID) ([]release.ReleaseModule, error)
	Update(ctx context.Context, module *release.ReleaseModule) error
	Delete(ctx context.Context, releaseID uuid.UUID, moduleKey string) error
	DeleteByRelease(ctx context.Context, releaseID uuid.UUID) error
}

// moduleRepositoryImpl implements ModuleRepository interface
type moduleRepositoryImpl struct {
	db *gorm.DB
}

// NewModuleRepository creates a new module repository instance
func NewModuleRepository(db *gorm.DB) ModuleRepository {
	return &moduleRepositoryImpl{db: db}
}

// Create creates a new release module
func (r *moduleRepositoryImpl) Create(ctx context.Context, module *release.ReleaseModule) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(module).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrModuleExists
		}
		return err
	}
	
	return nil
}

// CreateBulk creates multiple release modules
func (r *moduleRepositoryImpl) CreateBulk(ctx context.Context, modules []release.ReleaseModule) error {
	if len(modules) == 0 {
		return nil
	}
	
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(&modules).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrModuleExists
		}
		return err
	}
	
	return nil
}

// GetByID retrieves a release module by ID
func (r *moduleRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*release.ReleaseModule, error) {
	db := base.GetDB(ctx, r.db)
	
	var module release.ReleaseModule
	if err := db.Where("id = ?", id).First(&module).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModuleNotFound
		}
		return nil, err
	}
	
	return &module, nil
}

// GetByReleaseAndKey retrieves a release module by release ID and module key
func (r *moduleRepositoryImpl) GetByReleaseAndKey(ctx context.Context, releaseID uuid.UUID, moduleKey string) (*release.ReleaseModule, error) {
	db := base.GetDB(ctx, r.db)
	
	var module release.ReleaseModule
	if err := db.Where("release_id = ? AND module_key = ?", releaseID, moduleKey).First(&module).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModuleNotFound
		}
		return nil, err
	}
	
	return &module, nil
}

// ListByRelease retrieves all modules for a release
func (r *moduleRepositoryImpl) ListByRelease(ctx context.Context, releaseID uuid.UUID) ([]release.ReleaseModule, error) {
	db := base.GetDB(ctx, r.db)
	
	var modules []release.ReleaseModule
	if err := db.Where("release_id = ?", releaseID).Order("module_key").Find(&modules).Error; err != nil {
		return nil, err
	}
	
	return modules, nil
}

// Update updates a release module
func (r *moduleRepositoryImpl) Update(ctx context.Context, module *release.ReleaseModule) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Model(module).Where("id = ?", module.ID).Updates(module)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrModuleNotFound
	}
	
	return nil
}

// Delete deletes a release module by release ID and module key
func (r *moduleRepositoryImpl) Delete(ctx context.Context, releaseID uuid.UUID, moduleKey string) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Where("release_id = ? AND module_key = ?", releaseID, moduleKey).Delete(&release.ReleaseModule{})
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrModuleNotFound
	}
	
	return nil
}

// DeleteByRelease deletes all modules for a release
func (r *moduleRepositoryImpl) DeleteByRelease(ctx context.Context, releaseID uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Where("release_id = ?", releaseID).Delete(&release.ReleaseModule{}).Error; err != nil {
		return err
	}
	
	return nil
}
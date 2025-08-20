package release

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	model "github.com/input-output-hk/catalyst-forge/services/api/internal/models/release"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
)

var (
	ErrRenderedReleaseNotFound = errors.New("rendered release not found")
	ErrRenderedReleaseExists   = errors.New("rendered release already exists")
)

// RenderedRepository defines operations for RenderedRelease records
type RenderedRepository interface {
	Create(ctx context.Context, rr *model.RenderedRelease) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.RenderedRelease, error)
	GetByDeploymentID(ctx context.Context, deploymentID uuid.UUID) (*model.RenderedRelease, error)
	GetByOCIDigest(ctx context.Context, digest string) (*model.RenderedRelease, error)
	List(ctx context.Context, filter RenderedListFilter) ([]model.RenderedRelease, int64, error)
	Update(ctx context.Context, rr *model.RenderedRelease) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// RenderedListFilter contains filter parameters for listing rendered releases
type RenderedListFilter struct {
	ReleaseID     *uuid.UUID
	EnvironmentID *uuid.UUID
	DeploymentID  *uuid.UUID
	OCIDigest     *string
	OutputHash    *string
	Since         *time.Time
	Until         *time.Time
	Pagination    *base.Pagination
	Sort          *base.Sort
}

type renderedRepositoryImpl struct {
	db *gorm.DB
}

// NewRenderedRepository creates a new repository instance
func NewRenderedRepository(db *gorm.DB) RenderedRepository {
	return &renderedRepositoryImpl{db: db}
}

// Create inserts a new rendered release
func (r *renderedRepositoryImpl) Create(ctx context.Context, rr *model.RenderedRelease) error {
	db := base.GetDB(ctx, r.db)

	if err := db.Create(rr).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrRenderedReleaseExists
		}
		return err
	}
	return nil
}

// GetByID retrieves a rendered release by ID
func (r *renderedRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*model.RenderedRelease, error) {
	db := base.GetDB(ctx, r.db)

	var rr model.RenderedRelease
	if err := db.Where("id = ?", id).First(&rr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRenderedReleaseNotFound
		}
		return nil, err
	}
	return &rr, nil
}

// GetByDeploymentID retrieves a rendered release by unique deployment ID
func (r *renderedRepositoryImpl) GetByDeploymentID(ctx context.Context, deploymentID uuid.UUID) (*model.RenderedRelease, error) {
	db := base.GetDB(ctx, r.db)

	var rr model.RenderedRelease
	if err := db.Where("deployment_id = ?", deploymentID).First(&rr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRenderedReleaseNotFound
		}
		return nil, err
	}
	return &rr, nil
}

// GetByOCIDigest retrieves a rendered release by OCI digest
func (r *renderedRepositoryImpl) GetByOCIDigest(ctx context.Context, digest string) (*model.RenderedRelease, error) {
	db := base.GetDB(ctx, r.db)

	var rr model.RenderedRelease
	if err := db.Where("oci_digest = ?", digest).First(&rr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRenderedReleaseNotFound
		}
		return nil, err
	}
	return &rr, nil
}

// List retrieves rendered releases matching the provided filter
func (r *renderedRepositoryImpl) List(ctx context.Context, filter RenderedListFilter) ([]model.RenderedRelease, int64, error) {
	db := base.GetDB(ctx, r.db)

	query := db.Model(&model.RenderedRelease{})

	if filter.ReleaseID != nil {
		query = query.Where("release_id = ?", *filter.ReleaseID)
	}
	if filter.EnvironmentID != nil {
		query = query.Where("environment_id = ?", *filter.EnvironmentID)
	}
	if filter.DeploymentID != nil {
		query = query.Where("deployment_id = ?", *filter.DeploymentID)
	}
	if filter.OCIDigest != nil {
		query = query.Where("oci_digest = ?", *filter.OCIDigest)
	}
	if filter.OutputHash != nil {
		query = query.Where("output_hash = ?", *filter.OutputHash)
	}
	if filter.Since != nil {
		query = query.Where("created_at >= ?", *filter.Since)
	}
	if filter.Until != nil {
		query = query.Where("created_at <= ?", *filter.Until)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = base.ApplySort(query, filter.Sort, "created_at DESC")
	query = base.ApplyPagination(query, filter.Pagination)

	var items []model.RenderedRelease
	if err := query.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Update updates a rendered release
func (r *renderedRepositoryImpl) Update(ctx context.Context, rr *model.RenderedRelease) error {
	db := base.GetDB(ctx, r.db)

	result := db.Model(rr).Where("id = ?", rr.ID).Updates(rr)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRenderedReleaseNotFound
	}
	return nil
}

// Delete deletes a rendered release by ID
func (r *renderedRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)

	result := db.Where("id = ?", id).Delete(&model.RenderedRelease{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRenderedReleaseNotFound
	}
	return nil
}

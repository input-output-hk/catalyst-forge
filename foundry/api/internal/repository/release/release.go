package release

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/enums"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/release"
	base "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
)

var (
	ErrReleaseNotFound     = errors.New("release not found")
	ErrReleaseExists       = errors.New("release already exists")
	ErrReleaseSealed       = errors.New("release is sealed and cannot be modified")
	ErrReleaseReferenced   = errors.New("release is referenced by deployments")
)

// Repository defines the interface for release operations
type Repository interface {
	Create(ctx context.Context, rel *release.Release) error
	GetByID(ctx context.Context, id uuid.UUID) (*release.Release, error)
	GetByProjectAndKey(ctx context.Context, projectID uuid.UUID, key string) (*release.Release, error)
	GetByProjectAndTag(ctx context.Context, projectID uuid.UUID, tag string) (*release.Release, error)
	GetByOCIDigest(ctx context.Context, digest string) (*release.Release, error)
	List(ctx context.Context, filter ListFilter) ([]release.Release, int64, error)
	Update(ctx context.Context, rel *release.Release) error
	Delete(ctx context.Context, id uuid.UUID) error
	IsSealed(ctx context.Context, id uuid.UUID) (bool, error)
}

// ListFilter contains filter parameters for listing releases
type ListFilter struct {
	ProjectID    *uuid.UUID
	ReleaseKey   *string
	Status       *enums.ReleaseStatus
	OCIDigest    *string
	Tag          *string
	CreatedBy    *string
	Since        *time.Time
	Until        *time.Time
	Pagination   *base.Pagination
	Sort         *base.Sort
}

// repositoryImpl implements Repository interface
type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

// Create creates a new release
func (r *repositoryImpl) Create(ctx context.Context, rel *release.Release) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(rel).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrReleaseExists
		}
		return err
	}
	
	return nil
}

// GetByID retrieves a release by ID
func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*release.Release, error) {
	db := base.GetDB(ctx, r.db)
	
	var rel release.Release
	if err := db.Where("id = ?", id).First(&rel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReleaseNotFound
		}
		return nil, err
	}
	
	return &rel, nil
}

// GetByProjectAndKey retrieves a release by project ID and release key
func (r *repositoryImpl) GetByProjectAndKey(ctx context.Context, projectID uuid.UUID, key string) (*release.Release, error) {
	db := base.GetDB(ctx, r.db)
	
	var rel release.Release
	if err := db.Where("project_id = ? AND release_key = ?", projectID, key).First(&rel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReleaseNotFound
		}
		return nil, err
	}
	
	return &rel, nil
}

// GetByProjectAndTag retrieves a release by project ID and tag
func (r *repositoryImpl) GetByProjectAndTag(ctx context.Context, projectID uuid.UUID, tag string) (*release.Release, error) {
	db := base.GetDB(ctx, r.db)
	
	var rel release.Release
	if err := db.Where("project_id = ? AND tag = ?", projectID, tag).First(&rel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReleaseNotFound
		}
		return nil, err
	}
	
	return &rel, nil
}

// GetByOCIDigest retrieves a release by OCI digest
func (r *repositoryImpl) GetByOCIDigest(ctx context.Context, digest string) (*release.Release, error) {
	db := base.GetDB(ctx, r.db)
	
	var rel release.Release
	if err := db.Where("oci_digest = ?", digest).First(&rel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReleaseNotFound
		}
		return nil, err
	}
	
	return &rel, nil
}

// List retrieves releases with filters and pagination
func (r *repositoryImpl) List(ctx context.Context, filter ListFilter) ([]release.Release, int64, error) {
	db := base.GetDB(ctx, r.db)
	
	query := db.Model(&release.Release{})
	
	// Apply filters
	if filter.ProjectID != nil {
		query = query.Where("project_id = ?", *filter.ProjectID)
	}
	if filter.ReleaseKey != nil {
		query = query.Where("release_key = ?", *filter.ReleaseKey)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.OCIDigest != nil {
		query = query.Where("oci_digest = ?", *filter.OCIDigest)
	}
	if filter.Tag != nil {
		query = query.Where("tag = ?", *filter.Tag)
	}
	if filter.CreatedBy != nil {
		query = query.Where("created_by = ?", *filter.CreatedBy)
	}
	if filter.Since != nil {
		query = query.Where("created_at >= ?", *filter.Since)
	}
	if filter.Until != nil {
		query = query.Where("created_at <= ?", *filter.Until)
	}
	
	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// Apply sorting
	query = base.ApplySort(query, filter.Sort, "created_at DESC")
	
	// Apply pagination
	query = base.ApplyPagination(query, filter.Pagination)
	
	// Fetch results
	var releases []release.Release
	if err := query.Find(&releases).Error; err != nil {
		return nil, 0, err
	}
	
	return releases, total, nil
}

// Update updates a release
func (r *repositoryImpl) Update(ctx context.Context, rel *release.Release) error {
	db := base.GetDB(ctx, r.db)
	
	// Check if release is sealed
	sealed, err := r.IsSealed(ctx, rel.ID)
	if err != nil {
		return err
	}
	
	if sealed {
		// Only allow certain updates on sealed releases
		allowedUpdates := map[string]interface{}{
			"signed":               rel.Signed,
			"sig_issuer":           rel.SigIssuer,
			"sig_subject":          rel.SigSubject,
			"signature_verified_at": rel.SignatureVerifiedAt,
			"updated_at":           time.Now(),
		}
		
		result := db.Model(&release.Release{}).Where("id = ?", rel.ID).Updates(allowedUpdates)
		if result.Error != nil {
			return result.Error
		}
		
		if result.RowsAffected == 0 {
			return ErrReleaseNotFound
		}
		
		return nil
	}
	
	// Full update for non-sealed releases
	result := db.Model(rel).Where("id = ?", rel.ID).Updates(rel)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrReleaseNotFound
	}
	
	return nil
}

// Delete deletes a release by ID
func (r *repositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	// Check if release is referenced by deployments
	var count int64
	db.Table("deployment").Where("release_id = ?", id).Count(&count)
	if count > 0 {
		return ErrReleaseReferenced
	}
	
	// Delete in transaction to ensure all related records are removed
	return db.Transaction(func(tx *gorm.DB) error {
		// Delete related records first
		if err := tx.Where("release_id = ?", id).Delete(&release.ReleaseModule{}).Error; err != nil {
			return err
		}
		if err := tx.Where("release_id = ?", id).Delete(&release.ReleaseInjection{}).Error; err != nil {
			return err
		}
		if err := tx.Where("release_id = ?", id).Delete(&release.ReleaseArtifact{}).Error; err != nil {
			return err
		}
		
		// Delete the release
		result := tx.Where("id = ?", id).Delete(&release.Release{})
		if result.Error != nil {
			return result.Error
		}
		
		if result.RowsAffected == 0 {
			return ErrReleaseNotFound
		}
		
		return nil
	})
}

// IsSealed checks if a release is sealed
func (r *repositoryImpl) IsSealed(ctx context.Context, id uuid.UUID) (bool, error) {
	db := base.GetDB(ctx, r.db)
	
	var status enums.ReleaseStatus
	if err := db.Model(&release.Release{}).Where("id = ?", id).Select("status").Scan(&status).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrReleaseNotFound
		}
		return false, err
	}
	
	return status == enums.ReleaseStatusSealed, nil
}
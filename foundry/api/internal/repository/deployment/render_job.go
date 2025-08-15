package deployment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/deployment"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/enums"
	base "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
)

var (
	ErrRenderJobNotFound = errors.New("render job not found")
	ErrRenderJobExists   = errors.New("render job already exists for this deployment")
)

// RenderJobRepository defines the interface for render job operations
type RenderJobRepository interface {
	Create(ctx context.Context, job *deployment.RenderJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*deployment.RenderJob, error)
	GetByDeploymentID(ctx context.Context, deploymentID uuid.UUID) (*deployment.RenderJob, error)
	GetByOCIDigest(ctx context.Context, digest string) (*deployment.RenderJob, error)
	List(ctx context.Context, filter RenderJobListFilter) ([]deployment.RenderJob, int64, error)
	Update(ctx context.Context, job *deployment.RenderJob) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status enums.RenderJobStatus) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// RenderJobListFilter contains filter parameters for listing render jobs
type RenderJobListFilter struct {
	DeploymentID    *uuid.UUID
	Status          *enums.RenderJobStatus
	OCIDigest       *string
	RendererVersion *string
	Signed          *bool
	Pagination      *base.Pagination
	Sort            *base.Sort
}

// renderJobRepositoryImpl implements RenderJobRepository interface
type renderJobRepositoryImpl struct {
	db *gorm.DB
}

// NewRenderJobRepository creates a new render job repository instance
func NewRenderJobRepository(db *gorm.DB) RenderJobRepository {
	return &renderJobRepositoryImpl{db: db}
}

// Create creates a new render job
func (r *renderJobRepositoryImpl) Create(ctx context.Context, job *deployment.RenderJob) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(job).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrRenderJobExists
		}
		return err
	}
	
	return nil
}

// GetByID retrieves a render job by ID
func (r *renderJobRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*deployment.RenderJob, error) {
	db := base.GetDB(ctx, r.db)
	
	var job deployment.RenderJob
	if err := db.Where("id = ?", id).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRenderJobNotFound
		}
		return nil, err
	}
	
	return &job, nil
}

// GetByDeploymentID retrieves a render job by deployment ID (1:1 relationship)
func (r *renderJobRepositoryImpl) GetByDeploymentID(ctx context.Context, deploymentID uuid.UUID) (*deployment.RenderJob, error) {
	db := base.GetDB(ctx, r.db)
	
	var job deployment.RenderJob
	if err := db.Where("deployment_id = ?", deploymentID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRenderJobNotFound
		}
		return nil, err
	}
	
	return &job, nil
}

// GetByOCIDigest retrieves render jobs by OCI digest
func (r *renderJobRepositoryImpl) GetByOCIDigest(ctx context.Context, digest string) (*deployment.RenderJob, error) {
	db := base.GetDB(ctx, r.db)
	
	var job deployment.RenderJob
	if err := db.Where("oci_digest = ?", digest).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRenderJobNotFound
		}
		return nil, err
	}
	
	return &job, nil
}

// List retrieves render jobs with filters and pagination
func (r *renderJobRepositoryImpl) List(ctx context.Context, filter RenderJobListFilter) ([]deployment.RenderJob, int64, error) {
	db := base.GetDB(ctx, r.db)
	
	query := db.Model(&deployment.RenderJob{})
	
	// Apply filters
	if filter.DeploymentID != nil {
		query = query.Where("deployment_id = ?", *filter.DeploymentID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.OCIDigest != nil {
		query = query.Where("oci_digest = ?", *filter.OCIDigest)
	}
	if filter.RendererVersion != nil {
		query = query.Where("renderer_version = ?", *filter.RendererVersion)
	}
	if filter.Signed != nil {
		query = query.Where("signed = ?", *filter.Signed)
	}
	
	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// Apply sorting
	query = base.ApplySort(query, filter.Sort, "started_at DESC")
	
	// Apply pagination
	query = base.ApplyPagination(query, filter.Pagination)
	
	// Fetch results
	var jobs []deployment.RenderJob
	if err := query.Find(&jobs).Error; err != nil {
		return nil, 0, err
	}
	
	return jobs, total, nil
}

// Update updates a render job
func (r *renderJobRepositoryImpl) Update(ctx context.Context, job *deployment.RenderJob) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Model(job).Where("id = ?", job.ID).Updates(job)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrRenderJobNotFound
	}
	
	return nil
}

// UpdateStatus updates the status of a render job
func (r *renderJobRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status enums.RenderJobStatus) error {
	db := base.GetDB(ctx, r.db)
	
	updates := map[string]interface{}{
		"status": status,
	}
	
	// If status is terminal, set finished_at
	if status == enums.RenderJobStatusSuccess || status == enums.RenderJobStatusFailed || status == enums.RenderJobStatusCached {
		updates["finished_at"] = time.Now()
	}
	
	result := db.Model(&deployment.RenderJob{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrRenderJobNotFound
	}
	
	return nil
}

// Delete deletes a render job by ID
func (r *renderJobRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Where("id = ?", id).Delete(&deployment.RenderJob{})
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrRenderJobNotFound
	}
	
	return nil
}
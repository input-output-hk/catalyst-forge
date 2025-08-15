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
	ErrDeploymentNotFound = errors.New("deployment not found")
)

// Repository defines the interface for deployment operations
type Repository interface {
	Create(ctx context.Context, deployment *deployment.Deployment) error
	GetByID(ctx context.Context, id uuid.UUID) (*deployment.Deployment, error)
	GetWithRenderJob(ctx context.Context, id uuid.UUID) (*deployment.Deployment, error)
	List(ctx context.Context, filter ListFilter) ([]deployment.Deployment, int64, error)
	Update(ctx context.Context, deployment *deployment.Deployment) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status enums.DeploymentStatus, lastError *string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ListFilter contains filter parameters for listing deployments
type ListFilter struct {
	ReleaseID  *uuid.UUID
	EnvID      *uuid.UUID
	ProjectID  *uuid.UUID
	Status     *enums.DeploymentStatus
	CreatedBy  *string
	Since      *time.Time
	Until      *time.Time
	Pagination *base.Pagination
	Sort       *base.Sort
}

// repositoryImpl implements Repository interface
type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

// Create creates a new deployment
func (r *repositoryImpl) Create(ctx context.Context, d *deployment.Deployment) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(d).Error; err != nil {
		return err
	}
	
	return nil
}

// GetByID retrieves a deployment by ID
func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*deployment.Deployment, error) {
	db := base.GetDB(ctx, r.db)
	
	var d deployment.Deployment
	if err := db.Where("id = ?", id).First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}
	
	return &d, nil
}

// GetWithRenderJob retrieves a deployment with its render job
func (r *repositoryImpl) GetWithRenderJob(ctx context.Context, id uuid.UUID) (*deployment.Deployment, error) {
	db := base.GetDB(ctx, r.db)
	
	var d deployment.Deployment
	if err := db.Preload("RenderJob").Where("id = ?", id).First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}
	
	return &d, nil
}

// List retrieves deployments with filters and pagination
func (r *repositoryImpl) List(ctx context.Context, filter ListFilter) ([]deployment.Deployment, int64, error) {
	db := base.GetDB(ctx, r.db)
	
	query := db.Model(&deployment.Deployment{})
	
	// Apply filters
	if filter.ReleaseID != nil {
		query = query.Where("release_id = ?", *filter.ReleaseID)
	}
	if filter.EnvID != nil {
		query = query.Where("env_id = ?", *filter.EnvID)
	}
	if filter.ProjectID != nil {
		query = query.Where("project_id = ?", *filter.ProjectID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
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
	var deployments []deployment.Deployment
	if err := query.Find(&deployments).Error; err != nil {
		return nil, 0, err
	}
	
	return deployments, total, nil
}

// Update updates a deployment
func (r *repositoryImpl) Update(ctx context.Context, d *deployment.Deployment) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Model(d).Where("id = ?", d.ID).Updates(d)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrDeploymentNotFound
	}
	
	return nil
}

// UpdateStatus updates the status of a deployment
func (r *repositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status enums.DeploymentStatus, lastError *string) error {
	db := base.GetDB(ctx, r.db)
	
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	
	if lastError != nil {
		updates["last_error"] = *lastError
	} else {
		updates["last_error"] = nil
	}
	
	result := db.Model(&deployment.Deployment{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrDeploymentNotFound
	}
	
	return nil
}

// Delete deletes a deployment by ID
func (r *repositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	// Cascade delete will handle related records (render_job, gitops_changes)
	result := db.Where("id = ?", id).Delete(&deployment.Deployment{})
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrDeploymentNotFound
	}
	
	return nil
}
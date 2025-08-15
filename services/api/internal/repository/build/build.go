package build

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/build"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/enums"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
)

var (
	ErrBuildNotFound = errors.New("build not found")
)

// Repository defines the interface for build operations
type Repository interface {
	Create(ctx context.Context, build *build.Build) error
	GetByID(ctx context.Context, id uuid.UUID) (*build.Build, error)
	List(ctx context.Context, filter ListFilter) ([]build.Build, int64, error)
	Update(ctx context.Context, build *build.Build) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status enums.BuildStatus) error
}

// ListFilter contains filter parameters for listing builds
type ListFilter struct {
	TraceID       *uuid.UUID
	RepoID        *uuid.UUID
	ProjectID     *uuid.UUID
	CommitSHA     *string
	Branch        *string
	WorkflowRunID *string
	Status        *enums.BuildStatus
	Since         *time.Time
	Until         *time.Time
	Pagination    *base.Pagination
	Sort          *base.Sort
}

// repositoryImpl implements Repository interface
type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

// Create creates a new build
func (r *repositoryImpl) Create(ctx context.Context, b *build.Build) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(b).Error; err != nil {
		return err
	}
	
	return nil
}

// GetByID retrieves a build by ID
func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*build.Build, error) {
	db := base.GetDB(ctx, r.db)
	
	var b build.Build
	if err := db.Where("id = ?", id).First(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBuildNotFound
		}
		return nil, err
	}
	
	return &b, nil
}

// List retrieves builds with filters and pagination
func (r *repositoryImpl) List(ctx context.Context, filter ListFilter) ([]build.Build, int64, error) {
	db := base.GetDB(ctx, r.db)
	
	query := db.Model(&build.Build{})
	
	// Apply filters
	if filter.TraceID != nil {
		query = query.Where("trace_id = ?", *filter.TraceID)
	}
	if filter.RepoID != nil {
		query = query.Where("repo_id = ?", *filter.RepoID)
	}
	if filter.ProjectID != nil {
		query = query.Where("project_id = ?", *filter.ProjectID)
	}
	if filter.CommitSHA != nil {
		query = query.Where("commit_sha = ?", *filter.CommitSHA)
	}
	if filter.Branch != nil {
		query = query.Where("branch = ?", *filter.Branch)
	}
	if filter.WorkflowRunID != nil {
		query = query.Where("workflow_run_id = ?", *filter.WorkflowRunID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Since != nil {
		query = query.Where("started_at >= ?", *filter.Since)
	}
	if filter.Until != nil {
		query = query.Where("started_at <= ?", *filter.Until)
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
	var builds []build.Build
	if err := query.Find(&builds).Error; err != nil {
		return nil, 0, err
	}
	
	return builds, total, nil
}

// Update updates a build
func (r *repositoryImpl) Update(ctx context.Context, b *build.Build) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Model(b).Where("id = ?", b.ID).Updates(b)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrBuildNotFound
	}
	
	return nil
}

// UpdateStatus updates the status of a build
func (r *repositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status enums.BuildStatus) error {
	db := base.GetDB(ctx, r.db)
	
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	
	// If status is terminal, set finished_at
	if status == enums.BuildStatusSuccess || status == enums.BuildStatusFailed || status == enums.BuildStatusCanceled {
		updates["finished_at"] = time.Now()
	}
	
	result := db.Model(&build.Build{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrBuildNotFound
	}
	
	return nil
}
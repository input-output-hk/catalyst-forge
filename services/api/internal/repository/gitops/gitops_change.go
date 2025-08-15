package gitops

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/gitops"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
)

var (
	ErrGitOpsChangeNotFound = errors.New("gitops change not found")
)

// Repository defines the interface for gitops change operations
type Repository interface {
	Create(ctx context.Context, change *gitops.GitOpsChange) error
	GetByID(ctx context.Context, id uuid.UUID) (*gitops.GitOpsChange, error)
	GetByCommitSHA(ctx context.Context, commitSHA string) ([]gitops.GitOpsChange, error)
	ListByDeployment(ctx context.Context, deploymentID uuid.UUID) ([]gitops.GitOpsChange, error)
	List(ctx context.Context, filter ListFilter) ([]gitops.GitOpsChange, int64, error)
	Update(ctx context.Context, change *gitops.GitOpsChange) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ListFilter contains filter parameters for listing gitops changes
type ListFilter struct {
	DeploymentID  *uuid.UUID
	Repo          *string
	Branch        *string
	CommitSHA     *string
	PRNumber      *int
	PointerType   *string
	MergedAfter   *time.Time
	MergedBefore  *time.Time
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

// Create creates a new gitops change
func (r *repositoryImpl) Create(ctx context.Context, change *gitops.GitOpsChange) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(change).Error; err != nil {
		return err
	}
	
	return nil
}

// GetByID retrieves a gitops change by ID
func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*gitops.GitOpsChange, error) {
	db := base.GetDB(ctx, r.db)
	
	var change gitops.GitOpsChange
	if err := db.Where("id = ?", id).First(&change).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGitOpsChangeNotFound
		}
		return nil, err
	}
	
	return &change, nil
}

// GetByCommitSHA retrieves gitops changes by commit SHA
func (r *repositoryImpl) GetByCommitSHA(ctx context.Context, commitSHA string) ([]gitops.GitOpsChange, error) {
	db := base.GetDB(ctx, r.db)
	
	var changes []gitops.GitOpsChange
	if err := db.Where("commit_sha = ?", commitSHA).Order("created_at DESC").Find(&changes).Error; err != nil {
		return nil, err
	}
	
	return changes, nil
}

// ListByDeployment retrieves all gitops changes for a deployment
func (r *repositoryImpl) ListByDeployment(ctx context.Context, deploymentID uuid.UUID) ([]gitops.GitOpsChange, error) {
	db := base.GetDB(ctx, r.db)
	
	var changes []gitops.GitOpsChange
	if err := db.Where("deployment_id = ?", deploymentID).Order("created_at DESC").Find(&changes).Error; err != nil {
		return nil, err
	}
	
	return changes, nil
}

// List retrieves gitops changes with filters and pagination
func (r *repositoryImpl) List(ctx context.Context, filter ListFilter) ([]gitops.GitOpsChange, int64, error) {
	db := base.GetDB(ctx, r.db)
	
	query := db.Model(&gitops.GitOpsChange{})
	
	// Apply filters
	if filter.DeploymentID != nil {
		query = query.Where("deployment_id = ?", *filter.DeploymentID)
	}
	if filter.Repo != nil {
		query = query.Where("repo = ?", *filter.Repo)
	}
	if filter.Branch != nil {
		query = query.Where("branch = ?", *filter.Branch)
	}
	if filter.CommitSHA != nil {
		query = query.Where("commit_sha = ?", *filter.CommitSHA)
	}
	if filter.PRNumber != nil {
		query = query.Where("pr_number = ?", *filter.PRNumber)
	}
	if filter.PointerType != nil {
		query = query.Where("pointer_type = ?", *filter.PointerType)
	}
	if filter.MergedAfter != nil {
		query = query.Where("merged_at >= ?", *filter.MergedAfter)
	}
	if filter.MergedBefore != nil {
		query = query.Where("merged_at <= ?", *filter.MergedBefore)
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
	var changes []gitops.GitOpsChange
	if err := query.Find(&changes).Error; err != nil {
		return nil, 0, err
	}
	
	return changes, total, nil
}

// Update updates a gitops change
func (r *repositoryImpl) Update(ctx context.Context, change *gitops.GitOpsChange) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Model(change).Where("id = ?", change.ID).Updates(change)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrGitOpsChangeNotFound
	}
	
	return nil
}

// Delete deletes a gitops change by ID
func (r *repositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Where("id = ?", id).Delete(&gitops.GitOpsChange{})
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrGitOpsChangeNotFound
	}
	
	return nil
}
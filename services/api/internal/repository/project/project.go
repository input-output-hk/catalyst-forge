package project

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/project"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrProjectExists   = errors.New("project already exists")
)

// Repository defines the interface for project operations
type Repository interface {
	Create(ctx context.Context, proj *project.Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*project.Project, error)
	GetByRepoAndPath(ctx context.Context, repoID uuid.UUID, path string) (*project.Project, error)
	List(ctx context.Context, filter ListFilter) ([]project.Project, int64, error)
	Update(ctx context.Context, proj *project.Project) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ListFilter contains filter parameters for listing projects
type ListFilter struct {
	RepoID       *uuid.UUID
	Path         *string
	Slug         *string
	Status       *project.ProjectStatus
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

// Create creates a new project
func (r *repositoryImpl) Create(ctx context.Context, proj *project.Project) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(proj).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrProjectExists
		}
		return err
	}
	
	return nil
}

// GetByID retrieves a project by ID
func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*project.Project, error) {
	db := base.GetDB(ctx, r.db)
	
	var proj project.Project
	if err := db.Where("id = ?", id).First(&proj).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	
	return &proj, nil
}

// GetByRepoAndPath retrieves a project by repository ID and path
func (r *repositoryImpl) GetByRepoAndPath(ctx context.Context, repoID uuid.UUID, path string) (*project.Project, error) {
	db := base.GetDB(ctx, r.db)
	
	var proj project.Project
	if err := db.Where("repo_id = ? AND path = ?", repoID, path).First(&proj).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	
	return &proj, nil
}

// List retrieves projects with filters and pagination
func (r *repositoryImpl) List(ctx context.Context, filter ListFilter) ([]project.Project, int64, error) {
	db := base.GetDB(ctx, r.db)
	
	query := db.Model(&project.Project{})
	
	// Apply filters
	if filter.RepoID != nil {
		query = query.Where("repo_id = ?", *filter.RepoID)
	}
	if filter.Path != nil {
		query = query.Where("path = ?", *filter.Path)
	}
	if filter.Slug != nil {
		query = query.Where("slug LIKE ?", "%"+*filter.Slug+"%")
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
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
	var projects []project.Project
	if err := query.Find(&projects).Error; err != nil {
		return nil, 0, err
	}
	
	return projects, total, nil
}

// Update updates a project
func (r *repositoryImpl) Update(ctx context.Context, proj *project.Project) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Model(proj).Where("id = ?", proj.ID).Updates(proj)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	
	return nil
}

// Delete deletes a project by ID
func (r *repositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Where("id = ?", id).Delete(&project.Project{})
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	
	return nil
}
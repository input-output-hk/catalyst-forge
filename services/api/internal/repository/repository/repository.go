package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/repository"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
)

var (
	ErrRepositoryNotFound = errors.New("repository not found")
	ErrRepositoryExists   = errors.New("repository already exists")
)

// Repository defines the interface for repository operations
type Repository interface {
	Create(ctx context.Context, repo *repository.Repository) error
	GetByID(ctx context.Context, id uuid.UUID) (*repository.Repository, error)
	GetByHostOrgName(ctx context.Context, host, org, name string) (*repository.Repository, error)
	List(ctx context.Context, filter ListFilter) ([]repository.Repository, int64, error)
	Update(ctx context.Context, repo *repository.Repository) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ListFilter contains filter parameters for listing repositories
type ListFilter struct {
	Host         *string
	Org          *string
	Name         *string
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

// Create creates a new repository
func (r *repositoryImpl) Create(ctx context.Context, repo *repository.Repository) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(repo).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrRepositoryExists
		}
		return err
	}
	
	return nil
}

// GetByID retrieves a repository by ID
func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*repository.Repository, error) {
	db := base.GetDB(ctx, r.db)
	
	var repo repository.Repository
	if err := db.Where("id = ?", id).First(&repo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}
	
	return &repo, nil
}

// GetByHostOrgName retrieves a repository by host, org, and name
func (r *repositoryImpl) GetByHostOrgName(ctx context.Context, host, org, name string) (*repository.Repository, error) {
	db := base.GetDB(ctx, r.db)
	
	var repo repository.Repository
	if err := db.Where("host = ? AND org = ? AND name = ?", host, org, name).First(&repo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}
	
	return &repo, nil
}

// List retrieves repositories with filters and pagination
func (r *repositoryImpl) List(ctx context.Context, filter ListFilter) ([]repository.Repository, int64, error) {
	db := base.GetDB(ctx, r.db)
	
	query := db.Model(&repository.Repository{})
	
	// Apply filters
	if filter.Host != nil {
		query = query.Where("host = ?", *filter.Host)
	}
	if filter.Org != nil {
		query = query.Where("org = ?", *filter.Org)
	}
	if filter.Name != nil {
		query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
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
	var repos []repository.Repository
	if err := query.Find(&repos).Error; err != nil {
		return nil, 0, err
	}
	
	return repos, total, nil
}

// Update updates a repository
func (r *repositoryImpl) Update(ctx context.Context, repo *repository.Repository) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Model(repo).Where("id = ?", repo.ID).Updates(repo)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrRepositoryNotFound
	}
	
	return nil
}

// Delete deletes a repository by ID
func (r *repositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Where("id = ?", id).Delete(&repository.Repository{})
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrRepositoryNotFound
	}
	
	return nil
}
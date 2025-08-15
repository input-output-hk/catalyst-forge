package environment

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/environment"
	base "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
)

var (
	ErrEnvironmentNotFound = errors.New("environment not found")
	ErrEnvironmentExists   = errors.New("environment already exists")
	ErrEnvironmentReferenced = errors.New("environment is referenced by deployments")
)

// Repository defines the interface for environment operations
type Repository interface {
	Create(ctx context.Context, env *environment.Environment) error
	GetByID(ctx context.Context, id uuid.UUID) (*environment.Environment, error)
	GetByName(ctx context.Context, name string) (*environment.Environment, error)
	List(ctx context.Context, filter ListFilter) ([]environment.Environment, int64, error)
	Update(ctx context.Context, env *environment.Environment) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ListFilter contains filter parameters for listing environments
type ListFilter struct {
	Name         *string
	Cluster      *string
	ArgoProject  *string
	IsProtected  *bool
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

// Create creates a new environment
func (r *repositoryImpl) Create(ctx context.Context, env *environment.Environment) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(env).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrEnvironmentExists
		}
		return err
	}
	
	return nil
}

// GetByID retrieves an environment by ID
func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*environment.Environment, error) {
	db := base.GetDB(ctx, r.db)
	
	var env environment.Environment
	if err := db.Where("id = ?", id).First(&env).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEnvironmentNotFound
		}
		return nil, err
	}
	
	return &env, nil
}

// GetByName retrieves an environment by name
func (r *repositoryImpl) GetByName(ctx context.Context, name string) (*environment.Environment, error) {
	db := base.GetDB(ctx, r.db)
	
	var env environment.Environment
	if err := db.Where("name = ?", name).First(&env).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEnvironmentNotFound
		}
		return nil, err
	}
	
	return &env, nil
}

// List retrieves environments with filters and pagination
func (r *repositoryImpl) List(ctx context.Context, filter ListFilter) ([]environment.Environment, int64, error) {
	db := base.GetDB(ctx, r.db)
	
	query := db.Model(&environment.Environment{})
	
	// Apply filters
	if filter.Name != nil {
		query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Cluster != nil {
		query = query.Where("cluster = ?", *filter.Cluster)
	}
	if filter.ArgoProject != nil {
		query = query.Where("argo_project = ?", *filter.ArgoProject)
	}
	if filter.IsProtected != nil {
		query = query.Where("is_protected = ?", *filter.IsProtected)
	}
	
	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// Apply sorting
	query = base.ApplySort(query, filter.Sort, "name ASC")
	
	// Apply pagination
	query = base.ApplyPagination(query, filter.Pagination)
	
	// Fetch results
	var envs []environment.Environment
	if err := query.Find(&envs).Error; err != nil {
		return nil, 0, err
	}
	
	return envs, total, nil
}

// Update updates an environment
func (r *repositoryImpl) Update(ctx context.Context, env *environment.Environment) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Model(env).Where("id = ?", env.ID).Updates(env)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return ErrEnvironmentExists
		}
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrEnvironmentNotFound
	}
	
	return nil
}

// Delete deletes an environment by ID
func (r *repositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	// Check if environment is referenced by deployments
	var count int64
	db.Table("deployment").Where("env_id = ?", id).Count(&count)
	if count > 0 {
		return ErrEnvironmentReferenced
	}
	
	result := db.Where("id = ?", id).Delete(&environment.Environment{})
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrEnvironmentNotFound
	}
	
	return nil
}
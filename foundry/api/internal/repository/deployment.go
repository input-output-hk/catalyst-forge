package repository

import (
	"context"
	"errors"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"gorm.io/gorm"
)

// DeploymentRepository defines the interface for deployment operations
type DeploymentRepository interface {
	Create(ctx context.Context, deployment *models.ReleaseDeployment) error
	GetByID(ctx context.Context, id string) (*models.ReleaseDeployment, error)
	Update(ctx context.Context, deployment *models.ReleaseDeployment) error
	ListByReleaseID(ctx context.Context, releaseID string) ([]models.ReleaseDeployment, error)
	GetLatestByReleaseID(ctx context.Context, releaseID string) (*models.ReleaseDeployment, error)
}

// GormDeploymentRepository implements DeploymentRepository using GORM
type GormDeploymentRepository struct {
	db *gorm.DB
}

// NewDeploymentRepository creates a new DeploymentRepository
func NewDeploymentRepository(db *gorm.DB) DeploymentRepository {
	return &GormDeploymentRepository{db: db}
}

// Create adds a new deployment to the database
func (r *GormDeploymentRepository) Create(ctx context.Context, deployment *models.ReleaseDeployment) error {
	return r.db.WithContext(ctx).Create(deployment).Error
}

// GetByID retrieves a deployment by its ID
func (r *GormDeploymentRepository) GetByID(ctx context.Context, id string) (*models.ReleaseDeployment, error) {
	var deployment models.ReleaseDeployment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&deployment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deployment not found")
		}
		return nil, err
	}
	return &deployment, nil
}

// Update modifies an existing deployment
func (r *GormDeploymentRepository) Update(ctx context.Context, deployment *models.ReleaseDeployment) error {
	return r.db.WithContext(ctx).Save(deployment).Error
}

// ListByReleaseID retrieves all deployments for a specific release
func (r *GormDeploymentRepository) ListByReleaseID(ctx context.Context, releaseID string) ([]models.ReleaseDeployment, error) {
	var deployments []models.ReleaseDeployment
	if err := r.db.WithContext(ctx).Where("release_id = ?", releaseID).Order("timestamp DESC").Find(&deployments).Error; err != nil {
		return nil, err
	}
	return deployments, nil
}

// GetLatestByReleaseID retrieves the most recent deployment for a release
func (r *GormDeploymentRepository) GetLatestByReleaseID(ctx context.Context, releaseID string) (*models.ReleaseDeployment, error) {
	var deployment models.ReleaseDeployment
	if err := r.db.WithContext(ctx).Where("release_id = ?", releaseID).Order("timestamp DESC").First(&deployment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("no deployments found for release")
		}
		return nil, err
	}
	return &deployment, nil
}

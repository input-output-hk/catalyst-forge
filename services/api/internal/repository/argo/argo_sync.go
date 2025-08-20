package argo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/argo"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
)

// Repository defines the interface for gitops sync operations
type Repository interface {
	Create(ctx context.Context, sync *argo.GitOpsSync) error
	List(ctx context.Context, filter ListFilter) ([]argo.GitOpsSync, int64, error)
	GetLatestByApp(ctx context.Context, envID uuid.UUID, appName string) (*argo.GitOpsSync, error)
}

// ListFilter contains filter parameters for listing gitops syncs
type ListFilter struct {
	DeploymentID   *uuid.UUID
	EnvID          *uuid.UUID
	AppName        *string
	SyncStatus     *string
	HealthStatus   *string
	ObservedAfter  *time.Time
	ObservedBefore *time.Time
	Pagination     *base.Pagination
	Sort           *base.Sort
}

// repositoryImpl implements Repository interface
type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

// Create creates a new gitops sync record
func (r *repositoryImpl) Create(ctx context.Context, sync *argo.GitOpsSync) error {
	db := base.GetDB(ctx, r.db)

	if err := db.Create(sync).Error; err != nil {
		return err
	}

	return nil
}

// List retrieves gitops sync records with filters and pagination
func (r *repositoryImpl) List(ctx context.Context, filter ListFilter) ([]argo.GitOpsSync, int64, error) {
	db := base.GetDB(ctx, r.db)

	query := db.Model(&argo.GitOpsSync{})

	// Apply filters
	if filter.DeploymentID != nil {
		query = query.Where("deployment_id = ?", *filter.DeploymentID)
	}
	if filter.EnvID != nil {
		query = query.Where("env_id = ?", *filter.EnvID)
	}
	if filter.AppName != nil {
		query = query.Where("app_name = ?", *filter.AppName)
	}
	if filter.SyncStatus != nil {
		query = query.Where("sync_status = ?", *filter.SyncStatus)
	}
	if filter.HealthStatus != nil {
		query = query.Where("health_status = ?", *filter.HealthStatus)
	}
	if filter.ObservedAfter != nil {
		query = query.Where("observed_at >= ?", *filter.ObservedAfter)
	}
	if filter.ObservedBefore != nil {
		query = query.Where("observed_at <= ?", *filter.ObservedBefore)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	query = base.ApplySort(query, filter.Sort, "observed_at DESC")

	// Apply pagination
	query = base.ApplyPagination(query, filter.Pagination)

	// Fetch results
	var syncs []argo.GitOpsSync
	if err := query.Find(&syncs).Error; err != nil {
		return nil, 0, err
	}

	return syncs, total, nil
}

// GetLatestByApp retrieves the latest sync record for an app in an environment
func (r *repositoryImpl) GetLatestByApp(ctx context.Context, envID uuid.UUID, appName string) (*argo.GitOpsSync, error) {
	db := base.GetDB(ctx, r.db)

	var sync argo.GitOpsSync
	if err := db.Where("env_id = ? AND app_name = ?", envID, appName).
		Order("observed_at DESC").
		First(&sync).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not an error, just no records
		}
		return nil, err
	}

	return &sync, nil
}

package repository

import (
	"context"
	"errors"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"gorm.io/gorm"
)

// EventRepository defines the interface for deployment event operations
type EventRepository interface {
	AddEvent(ctx context.Context, event *models.DeploymentEvent) error
	ListEventsByDeploymentID(ctx context.Context, deploymentID string) ([]models.DeploymentEvent, error)
}

// GormEventRepository implements EventRepository using GORM
type GormEventRepository struct {
	db *gorm.DB
}

// NewEventRepository creates a new EventRepository
func NewEventRepository(db *gorm.DB) EventRepository {
	return &GormEventRepository{db: db}
}

// AddEvent adds a new event to the database
func (r *GormEventRepository) AddEvent(ctx context.Context, event *models.DeploymentEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return r.db.WithContext(ctx).Create(event).Error
}

// ListEventsByDeploymentID retrieves all events for a specific deployment
func (r *GormEventRepository) ListEventsByDeploymentID(ctx context.Context, deploymentID string) ([]models.DeploymentEvent, error) {
	var events []models.DeploymentEvent

	var count int64
	if err := r.db.WithContext(ctx).Model(&models.ReleaseDeployment{}).Where("id = ?", deploymentID).Count(&count).Error; err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, errors.New("deployment not found")
	}

	if err := r.db.WithContext(ctx).
		Where("deployment_id = ?", deploymentID).
		Order("timestamp DESC").
		Find(&events).Error; err != nil {
		return nil, err
	}

	return events, nil
}

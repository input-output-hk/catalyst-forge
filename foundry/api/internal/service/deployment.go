package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/k8s"
	"gorm.io/gorm"
)

// DeploymentService defines the interface for deployment-related business operations
type DeploymentService interface {
	CreateDeployment(ctx context.Context, releaseID string) (*models.ReleaseDeployment, error)
	GetDeployment(ctx context.Context, id string) (*models.ReleaseDeployment, error)
	UpdateDeployment(ctx context.Context, deployment *models.ReleaseDeployment) error
	ListDeployments(ctx context.Context, releaseID string) ([]models.ReleaseDeployment, error)
	GetLatestDeployment(ctx context.Context, releaseID string) (*models.ReleaseDeployment, error)

	// Event operations
	AddDeploymentEvent(ctx context.Context, deploymentID string, name string, message string) error
	GetDeploymentEvents(ctx context.Context, deploymentID string) ([]models.DeploymentEvent, error)
}

// DeploymentServiceImpl implements the DeploymentService interface
type DeploymentServiceImpl struct {
	deploymentRepo repository.DeploymentRepository
	releaseRepo    repository.ReleaseRepository
	eventRepo      repository.EventRepository
	k8sClient      k8s.Client
	logger         *slog.Logger
	db             *gorm.DB
}

// NewDeploymentService creates a new instance of DeploymentService
func NewDeploymentService(
	deploymentRepo repository.DeploymentRepository,
	releaseRepo repository.ReleaseRepository,
	eventRepo repository.EventRepository,
	k8sClient k8s.Client,
	db *gorm.DB,
	logger *slog.Logger,
) DeploymentService {
	return &DeploymentServiceImpl{
		deploymentRepo: deploymentRepo,
		releaseRepo:    releaseRepo,
		eventRepo:      eventRepo,
		k8sClient:      k8sClient,
		db:             db,
		logger:         logger,
	}
}

// CreateDeployment creates a new deployment for a release
func (s *DeploymentServiceImpl) CreateDeployment(ctx context.Context, releaseID string) (*models.ReleaseDeployment, error) {
	release, err := s.releaseRepo.GetByID(ctx, releaseID)
	if err != nil {
		return nil, err
	}

	var deployment *models.ReleaseDeployment
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		deploymentID := fmt.Sprintf("%s-%d", releaseID, now.UnixNano())

		deployment = &models.ReleaseDeployment{
			ID:        deploymentID,
			ReleaseID: releaseID,
			Timestamp: now,
			Status:    models.DeploymentStatusPending,
			Attempts:  0,
		}

		txDeploymentRepo := repository.NewDeploymentRepository(tx)
		if err := txDeploymentRepo.Create(ctx, deployment); err != nil {
			s.logger.Error("Failed to create deployment",
				"deploymentID", deployment.ID,
				"releaseID", releaseID,
				"error", err)
			return err
		}

		s.logger.Info("Creating Kubernetes deployment resource",
			"deploymentID", deployment.ID,
			"releaseID", releaseID)

		if err := s.k8sClient.CreateDeployment(ctx, deployment); err != nil {
			return fmt.Errorf("failed to create Kubernetes resource: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	deployment.Release = *release
	return deployment, nil
}

// GetDeployment retrieves a deployment by its ID
func (s *DeploymentServiceImpl) GetDeployment(ctx context.Context, id string) (*models.ReleaseDeployment, error) {
	deployment, err := s.deploymentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	release, err := s.releaseRepo.GetByID(ctx, deployment.ReleaseID)
	if err != nil {
		return nil, err
	}
	deployment.Release = *release

	events, err := s.eventRepo.ListEventsByDeploymentID(ctx, id)
	if err == nil {
		deployment.Events = events
	}

	return deployment, nil
}

// UpdateDeployment updates a deployment with new values
func (s *DeploymentServiceImpl) UpdateDeployment(ctx context.Context, deployment *models.ReleaseDeployment) error {
	existing, err := s.deploymentRepo.GetByID(ctx, deployment.ID)
	if err != nil {
		return err
	}

	deployment.CreatedAt = existing.CreatedAt

	return s.deploymentRepo.Update(ctx, deployment)
}

// ListDeployments retrieves all deployments for a specific release
func (s *DeploymentServiceImpl) ListDeployments(ctx context.Context, releaseID string) ([]models.ReleaseDeployment, error) {
	_, err := s.releaseRepo.GetByID(ctx, releaseID)
	if err != nil {
		return nil, err
	}

	return s.deploymentRepo.ListByReleaseID(ctx, releaseID)
}

// GetLatestDeployment retrieves the most recent deployment for a release
func (s *DeploymentServiceImpl) GetLatestDeployment(ctx context.Context, releaseID string) (*models.ReleaseDeployment, error) {
	_, err := s.releaseRepo.GetByID(ctx, releaseID)
	if err != nil {
		return nil, err
	}

	return s.deploymentRepo.GetLatestByReleaseID(ctx, releaseID)
}

// AddDeploymentEvent adds a new event to a deployment
func (s *DeploymentServiceImpl) AddDeploymentEvent(ctx context.Context, deploymentID string, name string, message string) error {
	_, err := s.deploymentRepo.GetByID(ctx, deploymentID)
	if err != nil {
		return err
	}

	event := &models.DeploymentEvent{
		DeploymentID: deploymentID,
		Name:         name,
		Message:      message,
		Timestamp:    time.Now(),
	}

	return s.eventRepo.AddEvent(ctx, event)
}

// GetDeploymentEvents retrieves all events for a deployment
func (s *DeploymentServiceImpl) GetDeploymentEvents(ctx context.Context, deploymentID string) ([]models.DeploymentEvent, error) {
	_, err := s.deploymentRepo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	return s.eventRepo.ListEventsByDeploymentID(ctx, deploymentID)
}

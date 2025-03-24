package service

import (
	"context"
	"fmt"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
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
}

// NewDeploymentService creates a new instance of DeploymentService
func NewDeploymentService(
	deploymentRepo repository.DeploymentRepository,
	releaseRepo repository.ReleaseRepository,
	eventRepo repository.EventRepository,
) DeploymentService {
	return &DeploymentServiceImpl{
		deploymentRepo: deploymentRepo,
		releaseRepo:    releaseRepo,
		eventRepo:      eventRepo,
	}
}

// CreateDeployment creates a new deployment for a release
func (s *DeploymentServiceImpl) CreateDeployment(ctx context.Context, releaseID string) (*models.ReleaseDeployment, error) {
	// Verify the release exists
	release, err := s.releaseRepo.GetByID(ctx, releaseID)
	if err != nil {
		return nil, err
	}

	// Create the deployment
	now := time.Now()
	deploymentID := fmt.Sprintf("%s-%d", releaseID, now.UnixNano())

	deployment := &models.ReleaseDeployment{
		ID:        deploymentID,
		ReleaseID: releaseID,
		Timestamp: now,
		Status:    models.DeploymentStatusPending,
		Attempts:  0,
	}

	if err := s.deploymentRepo.Create(ctx, deployment); err != nil {
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

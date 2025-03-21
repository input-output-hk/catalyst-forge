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
	UpdateDeploymentStatus(ctx context.Context, id string, status models.DeploymentStatus, reason string) error
	ListDeployments(ctx context.Context, releaseID string) ([]models.ReleaseDeployment, error)
	GetLatestDeployment(ctx context.Context, releaseID string) (*models.ReleaseDeployment, error)
}

// DeploymentServiceImpl implements the DeploymentService interface
type DeploymentServiceImpl struct {
	deploymentRepo repository.DeploymentRepository
	releaseRepo    repository.ReleaseRepository
}

// NewDeploymentService creates a new instance of DeploymentService
func NewDeploymentService(
	deploymentRepo repository.DeploymentRepository,
	releaseRepo repository.ReleaseRepository,
) DeploymentService {
	return &DeploymentServiceImpl{
		deploymentRepo: deploymentRepo,
		releaseRepo:    releaseRepo,
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

	return deployment, nil
}

// UpdateDeploymentStatus updates the status of a deployment
func (s *DeploymentServiceImpl) UpdateDeploymentStatus(ctx context.Context, id string, status models.DeploymentStatus, reason string) error {
	deployment, err := s.deploymentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	deployment.Status = status
	deployment.Reason = reason

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

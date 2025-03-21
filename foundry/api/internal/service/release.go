package service

import (
	"context"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
)

// ReleaseService defines the interface for release-related business operations
type ReleaseService interface {
	CreateRelease(ctx context.Context, release *models.Release) error
	GetRelease(ctx context.Context, id string) (*models.Release, error)
	UpdateRelease(ctx context.Context, release *models.Release) error
	DeleteRelease(ctx context.Context, id string) error
	ListReleases(ctx context.Context, projectName string) ([]models.Release, error)
	ListAllReleases(ctx context.Context) ([]models.Release, error)
	GetReleaseByAlias(ctx context.Context, aliasName string) (*models.Release, error)
	CreateReleaseAlias(ctx context.Context, aliasName string, releaseID string) error
	DeleteReleaseAlias(ctx context.Context, aliasName string) error
	ListReleaseAliases(ctx context.Context, releaseID string) ([]models.ReleaseAlias, error)
}

// ReleaseServiceImpl implements the ReleaseService interface
type ReleaseServiceImpl struct {
	releaseRepo    repository.ReleaseRepository
	aliasRepo      repository.AliasRepository
	counterRepo    repository.IDCounterRepository
	deploymentRepo repository.DeploymentRepository
}

// NewReleaseService creates a new instance of ReleaseService
func NewReleaseService(
	releaseRepo repository.ReleaseRepository,
	aliasRepo repository.AliasRepository,
	counterRepo repository.IDCounterRepository,
	deploymentRepo repository.DeploymentRepository,
) ReleaseService {
	return &ReleaseServiceImpl{
		releaseRepo:    releaseRepo,
		aliasRepo:      aliasRepo,
		counterRepo:    counterRepo,
		deploymentRepo: deploymentRepo,
	}
}

// CreateRelease creates a new release with a generated ID
func (s *ReleaseServiceImpl) CreateRelease(ctx context.Context, release *models.Release) error {
	// Generate the next ID for this project and branch combination
	nextID, err := s.counterRepo.GetNextID(ctx, release.Project, release.SourceBranch)
	if err != nil {
		return err
	}

	release.ID = nextID
	release.Created = time.Now()
	return s.releaseRepo.Create(ctx, release)
}

// GetRelease retrieves a release by its ID
func (s *ReleaseServiceImpl) GetRelease(ctx context.Context, id string) (*models.Release, error) {
	release, err := s.releaseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	deployments, err := s.deploymentRepo.ListByReleaseID(ctx, release.ID)
	if err != nil {
		return nil, err
	}
	release.Deployments = deployments

	return release, nil
}

// UpdateRelease updates an existing release
func (s *ReleaseServiceImpl) UpdateRelease(ctx context.Context, release *models.Release) error {
	existing, err := s.releaseRepo.GetByID(ctx, release.ID)
	if err != nil {
		return err
	}

	release.CreatedAt = existing.CreatedAt
	release.Created = existing.Created

	return s.releaseRepo.Update(ctx, release)
}

// DeleteRelease removes a release
func (s *ReleaseServiceImpl) DeleteRelease(ctx context.Context, id string) error {
	return s.releaseRepo.Delete(ctx, id)
}

// ListReleases retrieves all releases for a specific project
func (s *ReleaseServiceImpl) ListReleases(ctx context.Context, projectName string) ([]models.Release, error) {
	return s.releaseRepo.List(ctx, projectName)
}

// ListAllReleases retrieves all releases
func (s *ReleaseServiceImpl) ListAllReleases(ctx context.Context) ([]models.Release, error) {
	return s.releaseRepo.ListAll(ctx)
}

// GetReleaseByAlias retrieves a release by its alias name
func (s *ReleaseServiceImpl) GetReleaseByAlias(ctx context.Context, aliasName string) (*models.Release, error) {
	return s.releaseRepo.GetByAlias(ctx, aliasName)
}

// CreateReleaseAlias creates a new alias for a release
func (s *ReleaseServiceImpl) CreateReleaseAlias(ctx context.Context, aliasName string, releaseID string) error {
	_, err := s.releaseRepo.GetByID(ctx, releaseID)
	if err != nil {
		return err
	}

	existingAlias, err := s.aliasRepo.Get(ctx, aliasName)
	if err == nil && existingAlias != nil {
		existingAlias.ReleaseID = releaseID
		return s.aliasRepo.Update(ctx, existingAlias)
	} else if err != nil && err.Error() != "alias not found" {
		return err
	}

	alias := &models.ReleaseAlias{
		Name:      aliasName,
		ReleaseID: releaseID,
	}

	return s.aliasRepo.Create(ctx, alias)
}

// DeleteReleaseAlias removes an alias
func (s *ReleaseServiceImpl) DeleteReleaseAlias(ctx context.Context, aliasName string) error {
	return s.aliasRepo.Delete(ctx, aliasName)
}

// ListReleaseAliases retrieves all aliases for a specific release
func (s *ReleaseServiceImpl) ListReleaseAliases(ctx context.Context, releaseID string) ([]models.ReleaseAlias, error) {
	_, err := s.releaseRepo.GetByID(ctx, releaseID)
	if err != nil {
		return nil, err
	}

	return s.aliasRepo.ListByReleaseID(ctx, releaseID)
}

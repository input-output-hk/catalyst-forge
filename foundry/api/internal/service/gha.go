package service

import (
	"fmt"
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/gha_auth.go . GithubAuthService

// GithubAuthService defines the interface for GitHub Actions authentication service operations
type GithubAuthService interface {
	CreateAuth(auth *models.GithubRepositoryAuth) error
	GetAuthByID(id uint) (*models.GithubRepositoryAuth, error)
	GetAuthByRepository(repository string) (*models.GithubRepositoryAuth, error)
	UpdateAuth(auth *models.GithubRepositoryAuth) error
	DeleteAuth(id uint) error
	ListAuths() ([]models.GithubRepositoryAuth, error)
	GetPermissionsForRepository(repository string) ([]auth.Permission, error)
}

// DefaultGithubAuthService is the default implementation of GithubAuthService
type DefaultGithubAuthService struct {
	repo   repository.GithubAuthRepository
	logger *slog.Logger
}

// NewGithubAuthService creates a new GitHub Actions authentication service
func NewGithubAuthService(repo repository.GithubAuthRepository, logger *slog.Logger) *DefaultGithubAuthService {
	return &DefaultGithubAuthService{
		repo:   repo,
		logger: logger,
	}
}

// CreateAuth creates a new GitHub Actions authentication configuration
func (s *DefaultGithubAuthService) CreateAuth(auth *models.GithubRepositoryAuth) error {
	// Validate that the repository format is correct (owner/repo)
	if err := s.validateRepositoryFormat(auth.Repository); err != nil {
		return fmt.Errorf("invalid repository format: %w", err)
	}

	// Check if repository already exists
	existing, err := s.repo.GetByRepository(auth.Repository)
	if err == nil && existing != nil {
		return fmt.Errorf("authentication configuration already exists for repository: %s", auth.Repository)
	}

	s.logger.Info("Creating GHA authentication configuration",
		"repository", auth.Repository,
		"created_by", auth.CreatedBy)

	return s.repo.Create(auth)
}

// GetAuthByID retrieves a GitHub Actions authentication configuration by ID
func (s *DefaultGithubAuthService) GetAuthByID(id uint) (*models.GithubRepositoryAuth, error) {
	return s.repo.GetByID(id)
}

// GetAuthByRepository retrieves a GitHub Actions authentication configuration by repository name
func (s *DefaultGithubAuthService) GetAuthByRepository(repository string) (*models.GithubRepositoryAuth, error) {
	return s.repo.GetByRepository(repository)
}

// UpdateAuth updates an existing GitHub Actions authentication configuration
func (s *DefaultGithubAuthService) UpdateAuth(auth *models.GithubRepositoryAuth) error {
	// Validate that the repository format is correct
	if err := s.validateRepositoryFormat(auth.Repository); err != nil {
		return fmt.Errorf("invalid repository format: %w", err)
	}

	s.logger.Info("Updating GHA authentication configuration",
		"repository", auth.Repository,
		"updated_by", auth.UpdatedBy)

	return s.repo.Update(auth)
}

// DeleteAuth deletes a GitHub Actions authentication configuration
func (s *DefaultGithubAuthService) DeleteAuth(id uint) error {
	auth, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get auth configuration: %w", err)
	}

	s.logger.Info("Deleting GHA authentication configuration",
		"repository", auth.Repository)

	return s.repo.Delete(id)
}

// ListAuths retrieves all GitHub Actions authentication configurations
func (s *DefaultGithubAuthService) ListAuths() ([]models.GithubRepositoryAuth, error) {
	return s.repo.List()
}

// GetPermissionsForRepository retrieves the permissions for a specific repository
func (s *DefaultGithubAuthService) GetPermissionsForRepository(repository string) ([]auth.Permission, error) {
	return s.repo.GetPermissionsForRepository(repository)
}

// validateRepositoryFormat validates that the repository name follows the owner/repo format
func (s *DefaultGithubAuthService) validateRepositoryFormat(repository string) error {
	if repository == "" {
		return fmt.Errorf("repository name cannot be empty")
	}

	// Check if it contains exactly one slash (owner/repo format)
	count := 0
	for _, char := range repository {
		if char == '/' {
			count++
		}
	}

	if count != 1 {
		return fmt.Errorf("repository must be in format 'owner/repo', got: %s", repository)
	}

	return nil
}

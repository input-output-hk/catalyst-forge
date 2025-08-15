package environment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/environment"
	base "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	envRepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/environment"
)

var (
	ErrEnvironmentNotFound = errors.New("environment not found")
	ErrEnvironmentExists   = errors.New("environment already exists")
	ErrEnvironmentInUse    = errors.New("environment is referenced by deployments and cannot be deleted")
	ErrProtectedEnvironment = errors.New("operation not allowed on protected environment")
)

// Service defines the interface for environment business logic
type Service interface {
	Create(ctx context.Context, req CreateRequest) (*environment.Environment, error)
	GetByID(ctx context.Context, id uuid.UUID) (*environment.Environment, error)
	GetByName(ctx context.Context, name string) (*environment.Environment, error)
	List(ctx context.Context, filter ListFilter) ([]environment.Environment, int64, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*environment.Environment, error)
	Delete(ctx context.Context, id uuid.UUID) error
	IsProtected(ctx context.Context, id uuid.UUID) (bool, error)
}

// CreateRequest represents a request to create an environment
type CreateRequest struct {
	Name        string  `json:"name"`
	Cluster     string  `json:"cluster"`
	ArgoProject *string `json:"argo_project,omitempty"`
	IsProtected bool    `json:"is_protected"`
}

// UpdateRequest represents a request to update an environment
type UpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Cluster     *string `json:"cluster,omitempty"`
	ArgoProject *string `json:"argo_project,omitempty"`
	IsProtected *bool   `json:"is_protected,omitempty"`
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

// serviceImpl implements Service interface
type serviceImpl struct {
	txManager base.TxManager
	envRepo   envRepo.Repository
}

// NewService creates a new environment service
func NewService(
	txManager base.TxManager,
	envRepo envRepo.Repository,
) Service {
	return &serviceImpl{
		txManager: txManager,
		envRepo:   envRepo,
	}
}

// Create creates a new environment
func (s *serviceImpl) Create(ctx context.Context, req CreateRequest) (*environment.Environment, error) {
	// Validate environment name
	if err := s.validateEnvironmentName(req.Name); err != nil {
		return nil, err
	}
	
	// Create environment
	env := &environment.Environment{
		Name:        req.Name,
		Cluster:     req.Cluster,
		ArgoProject: req.ArgoProject,
		IsProtected: req.IsProtected,
	}
	
	if err := s.envRepo.Create(ctx, env); err != nil {
		if errors.Is(err, envRepo.ErrEnvironmentExists) {
			return nil, ErrEnvironmentExists
		}
		return nil, err
	}
	
	return env, nil
}

// GetByID retrieves an environment by ID
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*environment.Environment, error) {
	env, err := s.envRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, envRepo.ErrEnvironmentNotFound) {
			return nil, ErrEnvironmentNotFound
		}
		return nil, err
	}
	
	return env, nil
}

// GetByName retrieves an environment by name
func (s *serviceImpl) GetByName(ctx context.Context, name string) (*environment.Environment, error) {
	env, err := s.envRepo.GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, envRepo.ErrEnvironmentNotFound) {
			return nil, ErrEnvironmentNotFound
		}
		return nil, err
	}
	
	return env, nil
}

// List retrieves environments with filters
func (s *serviceImpl) List(ctx context.Context, filter ListFilter) ([]environment.Environment, int64, error) {
	repoFilter := envRepo.ListFilter{
		Name:        filter.Name,
		Cluster:     filter.Cluster,
		ArgoProject: filter.ArgoProject,
		IsProtected: filter.IsProtected,
		Pagination:  filter.Pagination,
		Sort:        filter.Sort,
	}
	
	return s.envRepo.List(ctx, repoFilter)
}

// Update updates an environment
func (s *serviceImpl) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*environment.Environment, error) {
	env, err := s.envRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, envRepo.ErrEnvironmentNotFound) {
			return nil, ErrEnvironmentNotFound
		}
		return nil, err
	}
	
	// Check if trying to modify protected environment settings
	if env.IsProtected && req.IsProtected != nil && !*req.IsProtected {
		// Trying to unprotect a protected environment - require additional validation
		// In a real system, this might require admin privileges
		// For now, we'll allow it but log it would be audited
	}
	
	// Apply updates
	if req.Name != nil {
		if err := s.validateEnvironmentName(*req.Name); err != nil {
			return nil, err
		}
		env.Name = *req.Name
	}
	
	if req.Cluster != nil {
		env.Cluster = *req.Cluster
	}
	
	if req.ArgoProject != nil {
		env.ArgoProject = req.ArgoProject
	}
	
	if req.IsProtected != nil {
		env.IsProtected = *req.IsProtected
	}
	
	if err := s.envRepo.Update(ctx, env); err != nil {
		if errors.Is(err, envRepo.ErrEnvironmentExists) {
			return nil, ErrEnvironmentExists
		}
		return nil, err
	}
	
	return env, nil
}

// Delete deletes an environment
func (s *serviceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if environment is protected
	env, err := s.envRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, envRepo.ErrEnvironmentNotFound) {
			return ErrEnvironmentNotFound
		}
		return err
	}
	
	if env.IsProtected {
		return ErrProtectedEnvironment
	}
	
	err = s.envRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, envRepo.ErrEnvironmentNotFound) {
			return ErrEnvironmentNotFound
		}
		if errors.Is(err, envRepo.ErrEnvironmentReferenced) {
			return ErrEnvironmentInUse
		}
		return err
	}
	
	return nil
}

// IsProtected checks if an environment is protected
func (s *serviceImpl) IsProtected(ctx context.Context, id uuid.UUID) (bool, error) {
	env, err := s.envRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, envRepo.ErrEnvironmentNotFound) {
			return false, ErrEnvironmentNotFound
		}
		return false, err
	}
	
	return env.IsProtected, nil
}

// validateEnvironmentName validates environment name
func (s *serviceImpl) validateEnvironmentName(name string) error {
	if name == "" {
		return fmt.Errorf("environment name cannot be empty")
	}
	
	if len(name) > 50 {
		return fmt.Errorf("environment name cannot exceed 50 characters")
	}
	
	// Could add more validation rules here (e.g., regex for allowed characters)
	
	return nil
}
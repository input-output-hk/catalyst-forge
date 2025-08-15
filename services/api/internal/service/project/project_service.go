package project

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/project"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	projectRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/project"
)

var (
	ErrProjectNotFound = errors.New("project not found")
)

// Service defines the interface for project business logic (read-only for now)
type Service interface {
	GetByID(ctx context.Context, id uuid.UUID) (*project.Project, error)
	GetByRepoAndPath(ctx context.Context, repoID uuid.UUID, path string) (*project.Project, error)
	List(ctx context.Context, filter ListFilter) ([]project.Project, int64, error)
}

// ListFilter contains filter parameters for listing projects
type ListFilter struct {
	RepoID     *uuid.UUID
	Path       *string
	Slug       *string
	Status     *project.ProjectStatus
	Pagination *base.Pagination
	Sort       *base.Sort
}

// serviceImpl implements Service interface
type serviceImpl struct {
	projectRepo projectRepo.Repository
}

// NewService creates a new project service
func NewService(projectRepo projectRepo.Repository) Service {
	return &serviceImpl{
		projectRepo: projectRepo,
	}
}

// GetByID retrieves a project by ID
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*project.Project, error) {
	p, err := s.projectRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, projectRepo.ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	
	return p, nil
}

// GetByRepoAndPath retrieves a project by repository ID and path
func (s *serviceImpl) GetByRepoAndPath(ctx context.Context, repoID uuid.UUID, path string) (*project.Project, error) {
	p, err := s.projectRepo.GetByRepoAndPath(ctx, repoID, path)
	if err != nil {
		if errors.Is(err, projectRepo.ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	
	return p, nil
}

// List retrieves projects with filters
func (s *serviceImpl) List(ctx context.Context, filter ListFilter) ([]project.Project, int64, error) {
	repoFilter := projectRepo.ListFilter{
		RepoID:     filter.RepoID,
		Path:       filter.Path,
		Slug:       filter.Slug,
		Status:     filter.Status,
		Pagination: filter.Pagination,
		Sort:       filter.Sort,
	}
	
	return s.projectRepo.List(ctx, repoFilter)
}
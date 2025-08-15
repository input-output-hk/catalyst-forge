package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/repository"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	repoRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/repository"
)

var (
	ErrRepositoryNotFound = errors.New("repository not found")
)

// Service defines the interface for repository business logic (read-only for now)
type Service interface {
	GetByID(ctx context.Context, id uuid.UUID) (*repository.Repository, error)
	GetByHostOrgName(ctx context.Context, host, org, name string) (*repository.Repository, error)
	List(ctx context.Context, filter ListFilter) ([]repository.Repository, int64, error)
}

// ListFilter contains filter parameters for listing repositories
type ListFilter struct {
	Host       *string
	Org        *string
	Name       *string
	Pagination *base.Pagination
	Sort       *base.Sort
}

// serviceImpl implements Service interface
type serviceImpl struct {
	repoRepo repoRepo.Repository
}

// NewService creates a new repository service
func NewService(repoRepo repoRepo.Repository) Service {
	return &serviceImpl{
		repoRepo: repoRepo,
	}
}

// GetByID retrieves a repository by ID
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*repository.Repository, error) {
	r, err := s.repoRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repoRepo.ErrRepositoryNotFound) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}
	
	return r, nil
}

// GetByHostOrgName retrieves a repository by host, org, and name
func (s *serviceImpl) GetByHostOrgName(ctx context.Context, host, org, name string) (*repository.Repository, error) {
	r, err := s.repoRepo.GetByHostOrgName(ctx, host, org, name)
	if err != nil {
		if errors.Is(err, repoRepo.ErrRepositoryNotFound) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}
	
	return r, nil
}

// List retrieves repositories with filters
func (s *serviceImpl) List(ctx context.Context, filter ListFilter) ([]repository.Repository, int64, error) {
	repoFilter := repoRepo.ListFilter{
		Host:       filter.Host,
		Org:        filter.Org,
		Name:       filter.Name,
		Pagination: filter.Pagination,
		Sort:       filter.Sort,
	}
	
	return s.repoRepo.List(ctx, repoFilter)
}
package build

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/build"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/enums"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	buildRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/build"
	projectRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/project"
	repoRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/repository"
)

var (
	ErrBuildNotFound      = errors.New("build not found")
	ErrProjectNotFound    = errors.New("project not found")
	ErrRepositoryNotFound = errors.New("repository not found")
	ErrInvalidStatus      = errors.New("invalid build status")
)

// Service defines the interface for build business logic
type Service interface {
	Create(ctx context.Context, req CreateRequest) (*build.Build, error)
	GetByID(ctx context.Context, id uuid.UUID) (*build.Build, error)
	List(ctx context.Context, filter ListFilter) ([]build.Build, int64, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*build.Build, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status enums.BuildStatus) error
}

// CreateRequest represents a request to create a build
type CreateRequest struct {
	TraceID       *uuid.UUID             `json:"trace_id,omitempty"`
	RepoID        uuid.UUID              `json:"repo_id"`
	ProjectID     uuid.UUID              `json:"project_id"`
	CommitSHA     string                 `json:"commit_sha"`
	Branch        *string                `json:"branch,omitempty"`
	WorkflowRunID *string                `json:"workflow_run_id,omitempty"`
	Status        enums.BuildStatus      `json:"status"`
	RunnerEnv     map[string]interface{} `json:"runner_env,omitempty"`
}

// UpdateRequest represents a request to update a build
type UpdateRequest struct {
	Status        *enums.BuildStatus     `json:"status,omitempty"`
	WorkflowRunID *string                `json:"workflow_run_id,omitempty"`
	RunnerEnv     map[string]interface{} `json:"runner_env,omitempty"`
	FinishedAt    *time.Time             `json:"finished_at,omitempty"`
}

// ListFilter contains filter parameters for listing builds
type ListFilter struct {
	TraceID       *uuid.UUID
	RepoID        *uuid.UUID
	ProjectID     *uuid.UUID
	CommitSHA     *string
	Branch        *string
	WorkflowRunID *string
	Status        *enums.BuildStatus
	Since         *time.Time
	Until         *time.Time
	Pagination    *base.Pagination
	Sort          *base.Sort
}

// serviceImpl implements Service interface
type serviceImpl struct {
	txManager   base.TxManager
	buildRepo   buildRepo.Repository
	projectRepo projectRepo.Repository
	repoRepo    repoRepo.Repository
}

// NewService creates a new build service
func NewService(
	txManager base.TxManager,
	buildRepo buildRepo.Repository,
	projectRepo projectRepo.Repository,
	repoRepo repoRepo.Repository,
) Service {
	return &serviceImpl{
		txManager:   txManager,
		buildRepo:   buildRepo,
		projectRepo: projectRepo,
		repoRepo:    repoRepo,
	}
}

// Create creates a new build
func (s *serviceImpl) Create(ctx context.Context, req CreateRequest) (*build.Build, error) {
	// Verify repository exists
	_, err := s.repoRepo.GetByID(ctx, req.RepoID)
	if err != nil {
		if errors.Is(err, repoRepo.ErrRepositoryNotFound) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}
	
	// Verify project exists
	_, err = s.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		if errors.Is(err, projectRepo.ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	
	// Create build
	b := &build.Build{
		TraceID:       req.TraceID,
		RepoID:        req.RepoID,
		ProjectID:     req.ProjectID,
		CommitSHA:     req.CommitSHA,
		Branch:        req.Branch,
		WorkflowRunID: req.WorkflowRunID,
		Status:        req.Status,
	}
	
	if req.RunnerEnv != nil {
		b.RunnerEnv = build.JSONB(req.RunnerEnv)
	}
	
	// Set finished_at if status is terminal
	if req.Status == enums.BuildStatusSuccess || req.Status == enums.BuildStatusFailed || req.Status == enums.BuildStatusCanceled {
		now := time.Now()
		b.FinishedAt = &now
	}
	
	if err := s.buildRepo.Create(ctx, b); err != nil {
		return nil, err
	}
	
	return b, nil
}

// GetByID retrieves a build by ID
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*build.Build, error) {
	b, err := s.buildRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, buildRepo.ErrBuildNotFound) {
			return nil, ErrBuildNotFound
		}
		return nil, err
	}
	
	return b, nil
}

// List retrieves builds with filters
func (s *serviceImpl) List(ctx context.Context, filter ListFilter) ([]build.Build, int64, error) {
	repoFilter := buildRepo.ListFilter{
		TraceID:       filter.TraceID,
		RepoID:        filter.RepoID,
		ProjectID:     filter.ProjectID,
		CommitSHA:     filter.CommitSHA,
		Branch:        filter.Branch,
		WorkflowRunID: filter.WorkflowRunID,
		Status:        filter.Status,
		Since:         filter.Since,
		Until:         filter.Until,
		Pagination:    filter.Pagination,
		Sort:          filter.Sort,
	}
	
	return s.buildRepo.List(ctx, repoFilter)
}

// Update updates a build
func (s *serviceImpl) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*build.Build, error) {
	b, err := s.buildRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, buildRepo.ErrBuildNotFound) {
			return nil, ErrBuildNotFound
		}
		return nil, err
	}
	
	// Apply updates
	if req.Status != nil {
		if err := s.validateStatusTransition(b.Status, *req.Status); err != nil {
			return nil, err
		}
		b.Status = *req.Status
		
		// Set finished_at if transitioning to terminal status
		if *req.Status == enums.BuildStatusSuccess || *req.Status == enums.BuildStatusFailed || *req.Status == enums.BuildStatusCanceled {
			if b.FinishedAt == nil {
				now := time.Now()
				b.FinishedAt = &now
			}
		}
	}
	
	if req.WorkflowRunID != nil {
		b.WorkflowRunID = req.WorkflowRunID
	}
	
	if req.RunnerEnv != nil {
		b.RunnerEnv = build.JSONB(req.RunnerEnv)
	}
	
	if req.FinishedAt != nil {
		b.FinishedAt = req.FinishedAt
	}
	
	if err := s.buildRepo.Update(ctx, b); err != nil {
		return nil, err
	}
	
	return b, nil
}

// UpdateStatus updates the status of a build
func (s *serviceImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status enums.BuildStatus) error {
	// Get current build to validate transition
	b, err := s.buildRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, buildRepo.ErrBuildNotFound) {
			return ErrBuildNotFound
		}
		return err
	}
	
	if err := s.validateStatusTransition(b.Status, status); err != nil {
		return err
	}
	
	return s.buildRepo.UpdateStatus(ctx, id, status)
}

// validateStatusTransition validates build status transitions
func (s *serviceImpl) validateStatusTransition(from, to enums.BuildStatus) error {
	// Define valid transitions
	validTransitions := map[enums.BuildStatus][]enums.BuildStatus{
		enums.BuildStatusQueued: {
			enums.BuildStatusRunning,
			enums.BuildStatusCanceled,
		},
		enums.BuildStatusRunning: {
			enums.BuildStatusSuccess,
			enums.BuildStatusFailed,
			enums.BuildStatusCanceled,
		},
		enums.BuildStatusSuccess: {
			// Terminal state
		},
		enums.BuildStatusFailed: {
			// Terminal state, but could allow retry
			enums.BuildStatusQueued,
		},
		enums.BuildStatusCanceled: {
			// Terminal state
		},
	}
	
	allowed, exists := validTransitions[from]
	if !exists {
		return errors.New("unknown build status")
	}
	
	for _, validTo := range allowed {
		if validTo == to {
			return nil
		}
	}
	
	return ErrInvalidStatus
}
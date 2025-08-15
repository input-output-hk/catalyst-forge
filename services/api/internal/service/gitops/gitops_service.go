package gitops

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/gitops"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	deploymentRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/deployment"
	gitopsRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/gitops"
)

var (
	ErrGitOpsChangeNotFound = errors.New("gitops change not found")
	ErrDeploymentNotFound   = errors.New("deployment not found")
	ErrInvalidPointerType   = errors.New("invalid pointer type")
)

// Service defines the interface for gitops change business logic
type Service interface {
	Create(ctx context.Context, req CreateRequest) (*gitops.GitOpsChange, error)
	GetByID(ctx context.Context, id uuid.UUID) (*gitops.GitOpsChange, error)
	GetByCommitSHA(ctx context.Context, commitSHA string) ([]gitops.GitOpsChange, error)
	ListByDeployment(ctx context.Context, deploymentID uuid.UUID) ([]gitops.GitOpsChange, error)
	List(ctx context.Context, filter ListFilter) ([]gitops.GitOpsChange, int64, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*gitops.GitOpsChange, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// CreateRequest represents a request to create a gitops change
type CreateRequest struct {
	DeploymentID  uuid.UUID `json:"deployment_id"`
	Repo          string    `json:"repo"`
	Branch        string    `json:"branch"`
	CommitSHA     string    `json:"commit_sha"`
	ChangePath    string    `json:"change_path"`
	PRNumber      *int      `json:"pr_number,omitempty"`
	PointerType   *string   `json:"pointer_type,omitempty"`
	PointerRef    *string   `json:"pointer_ref,omitempty"`
	PointerDigest *string   `json:"pointer_digest,omitempty"`
}

// UpdateRequest represents a request to update a gitops change
type UpdateRequest struct {
	PRNumber      *int       `json:"pr_number,omitempty"`
	MergedAt      *time.Time `json:"merged_at,omitempty"`
	PointerType   *string    `json:"pointer_type,omitempty"`
	PointerRef    *string    `json:"pointer_ref,omitempty"`
	PointerDigest *string    `json:"pointer_digest,omitempty"`
}

// ListFilter contains filter parameters for listing gitops changes
type ListFilter struct {
	DeploymentID  *uuid.UUID
	Repo          *string
	Branch        *string
	CommitSHA     *string
	PRNumber      *int
	PointerType   *string
	MergedAfter   *time.Time
	MergedBefore  *time.Time
	Pagination    *base.Pagination
	Sort          *base.Sort
}

// serviceImpl implements Service interface
type serviceImpl struct {
	txManager      base.TxManager
	gitopsRepo     gitopsRepo.Repository
	deploymentRepo deploymentRepo.Repository
}

// NewService creates a new gitops service
func NewService(
	txManager base.TxManager,
	gitopsRepo gitopsRepo.Repository,
	deploymentRepo deploymentRepo.Repository,
) Service {
	return &serviceImpl{
		txManager:      txManager,
		gitopsRepo:     gitopsRepo,
		deploymentRepo: deploymentRepo,
	}
}

// Create creates a new gitops change
func (s *serviceImpl) Create(ctx context.Context, req CreateRequest) (*gitops.GitOpsChange, error) {
	// Verify deployment exists
	_, err := s.deploymentRepo.GetByID(ctx, req.DeploymentID)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrDeploymentNotFound) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}
	
	// Validate pointer type if provided
	if req.PointerType != nil {
		if *req.PointerType != "release" && *req.PointerType != "rendered" {
			return nil, ErrInvalidPointerType
		}
	}
	
	// Create gitops change
	change := &gitops.GitOpsChange{
		DeploymentID:  req.DeploymentID,
		Repo:          req.Repo,
		Branch:        req.Branch,
		CommitSHA:     req.CommitSHA,
		ChangePath:    req.ChangePath,
		PRNumber:      req.PRNumber,
		PointerType:   (*gitops.PointerType)(req.PointerType),
		PointerRef:    req.PointerRef,
		PointerDigest: req.PointerDigest,
	}
	
	if err := s.gitopsRepo.Create(ctx, change); err != nil {
		return nil, err
	}
	
	return change, nil
}

// GetByID retrieves a gitops change by ID
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*gitops.GitOpsChange, error) {
	change, err := s.gitopsRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gitopsRepo.ErrGitOpsChangeNotFound) {
			return nil, ErrGitOpsChangeNotFound
		}
		return nil, err
	}
	
	return change, nil
}

// GetByCommitSHA retrieves gitops changes by commit SHA
func (s *serviceImpl) GetByCommitSHA(ctx context.Context, commitSHA string) ([]gitops.GitOpsChange, error) {
	return s.gitopsRepo.GetByCommitSHA(ctx, commitSHA)
}

// ListByDeployment retrieves all gitops changes for a deployment
func (s *serviceImpl) ListByDeployment(ctx context.Context, deploymentID uuid.UUID) ([]gitops.GitOpsChange, error) {
	// Verify deployment exists
	_, err := s.deploymentRepo.GetByID(ctx, deploymentID)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrDeploymentNotFound) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}
	
	return s.gitopsRepo.ListByDeployment(ctx, deploymentID)
}

// List retrieves gitops changes with filters
func (s *serviceImpl) List(ctx context.Context, filter ListFilter) ([]gitops.GitOpsChange, int64, error) {
	repoFilter := gitopsRepo.ListFilter{
		DeploymentID:  filter.DeploymentID,
		Repo:          filter.Repo,
		Branch:        filter.Branch,
		CommitSHA:     filter.CommitSHA,
		PRNumber:      filter.PRNumber,
		PointerType:   filter.PointerType,
		MergedAfter:   filter.MergedAfter,
		MergedBefore:  filter.MergedBefore,
		Pagination:    filter.Pagination,
		Sort:          filter.Sort,
	}
	
	return s.gitopsRepo.List(ctx, repoFilter)
}

// Update updates a gitops change
func (s *serviceImpl) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*gitops.GitOpsChange, error) {
	change, err := s.gitopsRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gitopsRepo.ErrGitOpsChangeNotFound) {
			return nil, ErrGitOpsChangeNotFound
		}
		return nil, err
	}
	
	// Validate pointer type if provided
	if req.PointerType != nil {
		if *req.PointerType != "release" && *req.PointerType != "rendered" {
			return nil, ErrInvalidPointerType
		}
		change.PointerType = (*gitops.PointerType)(req.PointerType)
	}
	
	// Apply updates
	if req.PRNumber != nil {
		change.PRNumber = req.PRNumber
	}
	
	if req.MergedAt != nil {
		change.MergedAt = req.MergedAt
	}
	
	if req.PointerRef != nil {
		change.PointerRef = req.PointerRef
	}
	
	if req.PointerDigest != nil {
		change.PointerDigest = req.PointerDigest
	}
	
	if err := s.gitopsRepo.Update(ctx, change); err != nil {
		return nil, err
	}
	
	return change, nil
}

// Delete deletes a gitops change
func (s *serviceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.gitopsRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, gitopsRepo.ErrGitOpsChangeNotFound) {
			return ErrGitOpsChangeNotFound
		}
		return err
	}
	
	return nil
}
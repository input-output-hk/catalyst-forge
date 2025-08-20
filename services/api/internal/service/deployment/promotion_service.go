package deployment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	depModel "github.com/input-output-hk/catalyst-forge/services/api/internal/models/deployment"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	depRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/deployment"
	envRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/environment"
	projRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/project"
	relRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/release"
)

// PromotionService defines business logic for promotions
type PromotionService interface {
	Create(ctx context.Context, req CreatePromotionRequest) (*depModel.Promotion, error)
	GetByID(ctx context.Context, id uuid.UUID) (*depModel.Promotion, error)
	List(ctx context.Context, filter PromotionListFilter) ([]depModel.Promotion, int64, error)
	Update(ctx context.Context, id uuid.UUID, req UpdatePromotionRequest) (*depModel.Promotion, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// CreatePromotionRequest request for creating promotion
type CreatePromotionRequest struct {
	ProjectID     uuid.UUID
	ReleaseID     uuid.UUID
	EnvironmentID uuid.UUID
	ApprovalMode  depModel.ApprovalMode
	RequestedBy   string
	Reason        *string
	PolicyResults map[string]interface{}
}

// UpdatePromotionRequest request for updating promotion
type UpdatePromotionRequest struct {
	Status           *depModel.PromotionStatus
	Reason           *string
	ApproverID       *string
	ApprovedAt       *time.Time
	StepUpVerifiedAt *time.Time
	PolicyResults    map[string]interface{}
	DeploymentID     *uuid.UUID
	TraceID          *uuid.UUID
}

// PromotionListFilter filters for listing promotions
type PromotionListFilter struct {
	ProjectID  *uuid.UUID
	EnvID      *uuid.UUID
	ReleaseID  *uuid.UUID
	Status     *depModel.PromotionStatus
	Pagination *base.Pagination
	Sort       *base.Sort
}

type promotionServiceImpl struct {
	txManager base.TxManager
	repo      depRepo.PromotionRepository
	project   projRepo.Repository
	release   relRepo.Repository
	env       envRepo.Repository
}

// NewPromotionService creates a new promotion service
func NewPromotionService(txManager base.TxManager, repo depRepo.PromotionRepository, project projRepo.Repository, release relRepo.Repository, env envRepo.Repository) PromotionService {
	return &promotionServiceImpl{txManager: txManager, repo: repo, project: project, release: release, env: env}
}

// Create creates a new promotion
func (s *promotionServiceImpl) Create(ctx context.Context, req CreatePromotionRequest) (*depModel.Promotion, error) {
	// Verify foreign keys exist
	if _, err := s.project.GetByID(ctx, req.ProjectID); err != nil {
		if errors.Is(err, projRepo.ErrProjectNotFound) {
			return nil, projRepo.ErrProjectNotFound
		}
		return nil, err
	}
	if _, err := s.release.GetByID(ctx, req.ReleaseID); err != nil {
		if errors.Is(err, relRepo.ErrReleaseNotFound) {
			return nil, relRepo.ErrReleaseNotFound
		}
		return nil, err
	}
	if _, err := s.env.GetByID(ctx, req.EnvironmentID); err != nil {
		if errors.Is(err, envRepo.ErrEnvironmentNotFound) {
			return nil, envRepo.ErrEnvironmentNotFound
		}
		return nil, err
	}

	p := &depModel.Promotion{
		ProjectID:    req.ProjectID,
		ReleaseID:    req.ReleaseID,
		EnvID:        req.EnvironmentID,
		Status:       depModel.PromotionStatusRequested,
		ApprovalMode: req.ApprovalMode,
		RequestedBy:  req.RequestedBy,
		RequestedAt:  time.Now(),
	}
	if req.Reason != nil {
		p.Reason = req.Reason
	}
	if req.PolicyResults != nil {
		p.PolicyResults = depModel.JSONB(req.PolicyResults)
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// GetByID gets promotion by id
func (s *promotionServiceImpl) GetByID(ctx context.Context, id uuid.UUID) (*depModel.Promotion, error) {
	return s.repo.GetByID(ctx, id)
}

// List lists promotions
func (s *promotionServiceImpl) List(ctx context.Context, filter PromotionListFilter) ([]depModel.Promotion, int64, error) {
	repoFilter := depRepo.PromotionListFilter{
		ProjectID:  filter.ProjectID,
		EnvID:      filter.EnvID,
		ReleaseID:  filter.ReleaseID,
		Status:     filter.Status,
		Pagination: filter.Pagination,
		Sort:       filter.Sort,
	}
	return s.repo.List(ctx, repoFilter)
}

// Update updates a promotion
func (s *promotionServiceImpl) Update(ctx context.Context, id uuid.UUID, req UpdatePromotionRequest) (*depModel.Promotion, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Status != nil {
		p.Status = *req.Status
	}
	if req.Reason != nil {
		p.Reason = req.Reason
	}
	if req.ApproverID != nil {
		p.ApproverID = req.ApproverID
	}
	if req.ApprovedAt != nil {
		p.ApprovedAt = req.ApprovedAt
	}
	if req.StepUpVerifiedAt != nil {
		p.StepUpVerifiedAt = req.StepUpVerifiedAt
	}
	if req.PolicyResults != nil {
		p.PolicyResults = depModel.JSONB(req.PolicyResults)
	}
	if req.DeploymentID != nil {
		p.DeploymentID = req.DeploymentID
	}
	if req.TraceID != nil {
		p.TraceID = req.TraceID
	}

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Delete deletes a promotion
func (s *promotionServiceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// helpers
func (s *promotionServiceImpl) formatNotFound(name string) error {
	return fmt.Errorf("%s not found", name)
}

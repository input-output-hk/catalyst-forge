package release

import (
	"context"
	"time"

	"github.com/google/uuid"

	model "github.com/input-output-hk/catalyst-forge/services/api/internal/models/release"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	renderedRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/release"
)

// RenderedService defines business logic for rendered releases
type RenderedService interface {
	Create(ctx context.Context, req RenderedCreateRequest) (*model.RenderedRelease, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.RenderedRelease, error)
	GetByDeployment(ctx context.Context, deploymentID uuid.UUID) (*model.RenderedRelease, error)
	List(ctx context.Context, filter RenderedListFilter) ([]model.RenderedRelease, int64, error)
	Update(ctx context.Context, id uuid.UUID, req RenderedUpdateRequest) (*model.RenderedRelease, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// RenderedCreateRequest request for creating rendered release
type RenderedCreateRequest struct {
	DeploymentID        uuid.UUID
	ReleaseID           uuid.UUID
	EnvironmentID       uuid.UUID
	RendererVersion     string
	ModuleVersions      []map[string]interface{}
	BundleHash          string
	OutputHash          string
	OCIRef              string
	OCIDigest           string
	StorageURI          *string
	Signed              *bool
	SignatureVerifiedAt *time.Time
}

// RenderedUpdateRequest request for updating rendered release
type RenderedUpdateRequest struct {
	OCIRef              *string
	OCIDigest           *string
	StorageURI          *string
	Signed              *bool
	SignatureVerifiedAt *time.Time
}

// RenderedListFilter filters for listing rendered releases
type RenderedListFilter struct {
	ReleaseID     *uuid.UUID
	EnvironmentID *uuid.UUID
	DeploymentID  *uuid.UUID
	OCIDigest     *string
	OutputHash    *string
	Pagination    *base.Pagination
	Sort          *base.Sort
}

type renderedServiceImpl struct {
	txManager base.TxManager
	repo      renderedRepo.RenderedRepository
}

// NewRenderedService creates a new rendered release service
func NewRenderedService(txManager base.TxManager, repo renderedRepo.RenderedRepository) RenderedService {
	return &renderedServiceImpl{txManager: txManager, repo: repo}
}

// Create creates a new rendered release
func (s *renderedServiceImpl) Create(ctx context.Context, req RenderedCreateRequest) (*model.RenderedRelease, error) {
	rr := &model.RenderedRelease{
		DeploymentID:    req.DeploymentID,
		ReleaseID:       req.ReleaseID,
		EnvironmentID:   req.EnvironmentID,
		RendererVersion: req.RendererVersion,
		ModuleVersions:  nil,
		BundleHash:      req.BundleHash,
		OutputHash:      req.OutputHash,
		OCIRef:          req.OCIRef,
		OCIDigest:       req.OCIDigest,
		StorageURI:      req.StorageURI,
	}
	if req.Signed != nil {
		rr.Signed = *req.Signed
	}
	if req.SignatureVerifiedAt != nil {
		rr.SignatureVerifiedAt = req.SignatureVerifiedAt
	}

	// Convert ModuleVersions if provided
	if req.ModuleVersions != nil {
		// store as datatypes.JSON via gorm; leave nil to use default [] from DB
		// here we rely on API layer to pass down as map, repository will persist
	}

	if err := s.repo.Create(ctx, rr); err != nil {
		return nil, err
	}
	return rr, nil
}

// GetByID returns a rendered release by id
func (s *renderedServiceImpl) GetByID(ctx context.Context, id uuid.UUID) (*model.RenderedRelease, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByDeployment returns a rendered release by deployment id
func (s *renderedServiceImpl) GetByDeployment(ctx context.Context, deploymentID uuid.UUID) (*model.RenderedRelease, error) {
	return s.repo.GetByDeploymentID(ctx, deploymentID)
}

// List returns rendered releases with filters
func (s *renderedServiceImpl) List(ctx context.Context, filter RenderedListFilter) ([]model.RenderedRelease, int64, error) {
	repoFilter := renderedRepo.RenderedListFilter{
		ReleaseID:     filter.ReleaseID,
		EnvironmentID: filter.EnvironmentID,
		DeploymentID:  filter.DeploymentID,
		OCIDigest:     filter.OCIDigest,
		OutputHash:    filter.OutputHash,
		Pagination:    filter.Pagination,
		Sort:          filter.Sort,
	}
	return s.repo.List(ctx, repoFilter)
}

// Update updates a rendered release
func (s *renderedServiceImpl) Update(ctx context.Context, id uuid.UUID, req RenderedUpdateRequest) (*model.RenderedRelease, error) {
	rr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.OCIRef != nil {
		rr.OCIRef = *req.OCIRef
	}
	if req.OCIDigest != nil {
		rr.OCIDigest = *req.OCIDigest
	}
	if req.StorageURI != nil {
		rr.StorageURI = req.StorageURI
	}
	if req.Signed != nil {
		rr.Signed = *req.Signed
	}
	if req.SignatureVerifiedAt != nil {
		rr.SignatureVerifiedAt = req.SignatureVerifiedAt
	}

	if err := s.repo.Update(ctx, rr); err != nil {
		return nil, err
	}
	return rr, nil
}

// Delete deletes a rendered release by ID
func (s *renderedServiceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

package render

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/deployment"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/enums"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	deploymentRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/deployment"
)

var (
	ErrRenderJobNotFound = errors.New("render job not found")
	ErrRenderJobExists   = errors.New("render job already exists for this deployment")
	ErrDeploymentNotFound = errors.New("deployment not found")
)

// Service defines the interface for render job business logic
type Service interface {
	Create(ctx context.Context, req CreateRequest) (*deployment.RenderJob, error)
	GetByID(ctx context.Context, id uuid.UUID) (*deployment.RenderJob, error)
	GetByDeploymentID(ctx context.Context, deploymentID uuid.UUID) (*deployment.RenderJob, error)
	List(ctx context.Context, filter ListFilter) ([]deployment.RenderJob, int64, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*deployment.RenderJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status enums.RenderJobStatus) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CreateRequest represents a request to create a render job
type CreateRequest struct {
	DeploymentID    uuid.UUID       `json:"deployment_id"`
	ModuleVersions  []ModuleVersion `json:"module_versions,omitempty"`
	BundleHash      *string         `json:"bundle_hash,omitempty"`
	RendererVersion *string         `json:"renderer_version,omitempty"`
}

// ModuleVersion represents a module version
type ModuleVersion struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// UpdateRequest represents a request to update a render job
type UpdateRequest struct {
	Status              *enums.RenderJobStatus `json:"status,omitempty"`
	ModuleVersions      []ModuleVersion        `json:"module_versions,omitempty"`
	BundleHash          *string                `json:"bundle_hash,omitempty"`
	OutputHash          *string                `json:"output_hash,omitempty"`
	StorageURI          *string                `json:"storage_uri,omitempty"`
	RendererVersion     *string                `json:"renderer_version,omitempty"`
	OCIRef              *string                `json:"oci_ref,omitempty"`
	OCIDigest           *string                `json:"oci_digest,omitempty"`
	Signed              *bool                  `json:"signed,omitempty"`
	SignatureVerifiedAt *time.Time             `json:"signature_verified_at,omitempty"`
	FinishedAt          *time.Time             `json:"finished_at,omitempty"`
}

// ListFilter contains filter parameters for listing render jobs
type ListFilter struct {
	DeploymentID    *uuid.UUID
	Status          *enums.RenderJobStatus
	OCIDigest       *string
	RendererVersion *string
	Signed          *bool
	Pagination      *base.Pagination
	Sort            *base.Sort
}

// serviceImpl implements Service interface
type serviceImpl struct {
	txManager        base.TxManager
	renderJobRepo    deploymentRepo.RenderJobRepository
	deploymentRepo   deploymentRepo.Repository
}

// NewService creates a new render job service
func NewService(
	txManager base.TxManager,
	renderJobRepo deploymentRepo.RenderJobRepository,
	deploymentRepo deploymentRepo.Repository,
) Service {
	return &serviceImpl{
		txManager:      txManager,
		renderJobRepo:  renderJobRepo,
		deploymentRepo: deploymentRepo,
	}
}

// Create creates a new render job (1:1 with deployment)
func (s *serviceImpl) Create(ctx context.Context, req CreateRequest) (*deployment.RenderJob, error) {
	// Verify deployment exists
	_, err := s.deploymentRepo.GetByID(ctx, req.DeploymentID)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrDeploymentNotFound) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}
	
	// Check if render job already exists for this deployment
	existing, err := s.renderJobRepo.GetByDeploymentID(ctx, req.DeploymentID)
	if err == nil && existing != nil {
		return nil, ErrRenderJobExists
	}
	
	// Create render job
	job := &deployment.RenderJob{
		DeploymentID:    req.DeploymentID,
		Status:          enums.RenderJobStatusPending,
		BundleHash:      req.BundleHash,
		RendererVersion: req.RendererVersion,
	}
	
	if len(req.ModuleVersions) > 0 {
		moduleVersionsJSON := make([]interface{}, len(req.ModuleVersions))
		for i, mv := range req.ModuleVersions {
			moduleVersionsJSON[i] = map[string]interface{}{
				"name":    mv.Name,
				"version": mv.Version,
			}
		}
		job.ModuleVersions = deployment.JSONB{"modules": moduleVersionsJSON}
	}
	
	if err := s.renderJobRepo.Create(ctx, job); err != nil {
		if errors.Is(err, deploymentRepo.ErrRenderJobExists) {
			return nil, ErrRenderJobExists
		}
		return nil, err
	}
	
	// Update deployment status to rendering
	_ = s.deploymentRepo.UpdateStatus(ctx, req.DeploymentID, enums.DeploymentStatusRendered, nil)
	
	return job, nil
}

// GetByID retrieves a render job by ID
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*deployment.RenderJob, error) {
	job, err := s.renderJobRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrRenderJobNotFound) {
			return nil, ErrRenderJobNotFound
		}
		return nil, err
	}
	
	return job, nil
}

// GetByDeploymentID retrieves a render job by deployment ID
func (s *serviceImpl) GetByDeploymentID(ctx context.Context, deploymentID uuid.UUID) (*deployment.RenderJob, error) {
	job, err := s.renderJobRepo.GetByDeploymentID(ctx, deploymentID)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrRenderJobNotFound) {
			return nil, ErrRenderJobNotFound
		}
		return nil, err
	}
	
	return job, nil
}

// List retrieves render jobs with filters
func (s *serviceImpl) List(ctx context.Context, filter ListFilter) ([]deployment.RenderJob, int64, error) {
	repoFilter := deploymentRepo.RenderJobListFilter{
		DeploymentID:    filter.DeploymentID,
		Status:          filter.Status,
		OCIDigest:       filter.OCIDigest,
		RendererVersion: filter.RendererVersion,
		Signed:          filter.Signed,
		Pagination:      filter.Pagination,
		Sort:            filter.Sort,
	}
	
	return s.renderJobRepo.List(ctx, repoFilter)
}

// Update updates a render job
func (s *serviceImpl) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*deployment.RenderJob, error) {
	job, err := s.renderJobRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrRenderJobNotFound) {
			return nil, ErrRenderJobNotFound
		}
		return nil, err
	}
	
	// Apply updates
	if req.Status != nil {
		job.Status = *req.Status
	}
	
	if len(req.ModuleVersions) > 0 {
		moduleVersionsJSON := make([]interface{}, len(req.ModuleVersions))
		for i, mv := range req.ModuleVersions {
			moduleVersionsJSON[i] = map[string]interface{}{
				"name":    mv.Name,
				"version": mv.Version,
			}
		}
		job.ModuleVersions = deployment.JSONB{"modules": moduleVersionsJSON}
	}
	
	if req.BundleHash != nil {
		job.BundleHash = req.BundleHash
	}
	
	if req.OutputHash != nil {
		job.OutputHash = req.OutputHash
	}
	
	if req.StorageURI != nil {
		job.StorageURI = req.StorageURI
	}
	
	if req.RendererVersion != nil {
		job.RendererVersion = req.RendererVersion
	}
	
	if req.OCIRef != nil {
		job.OCIRef = req.OCIRef
	}
	
	if req.OCIDigest != nil {
		job.OCIDigest = req.OCIDigest
	}
	
	if req.Signed != nil {
		job.Signed = *req.Signed
	}
	
	if req.SignatureVerifiedAt != nil {
		job.SignatureVerifiedAt = req.SignatureVerifiedAt
	}
	
	if req.FinishedAt != nil {
		job.FinishedAt = req.FinishedAt
	}
	
	if err := s.renderJobRepo.Update(ctx, job); err != nil {
		return nil, err
	}
	
	// Update deployment status based on render job status
	if req.Status != nil {
		switch *req.Status {
		case enums.RenderJobStatusSuccess:
			_ = s.deploymentRepo.UpdateStatus(ctx, job.DeploymentID, enums.DeploymentStatusPushed, nil)
		case enums.RenderJobStatusFailed:
			errorMsg := "render job failed"
			_ = s.deploymentRepo.UpdateStatus(ctx, job.DeploymentID, enums.DeploymentStatusFailed, &errorMsg)
		}
	}
	
	return job, nil
}

// UpdateStatus updates the status of a render job
func (s *serviceImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status enums.RenderJobStatus) error {
	err := s.renderJobRepo.UpdateStatus(ctx, id, status)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrRenderJobNotFound) {
			return ErrRenderJobNotFound
		}
		return err
	}
	
	// Get the render job to update deployment status
	job, err := s.renderJobRepo.GetByID(ctx, id)
	if err == nil {
		switch status {
		case enums.RenderJobStatusSuccess:
			_ = s.deploymentRepo.UpdateStatus(ctx, job.DeploymentID, enums.DeploymentStatusPushed, nil)
		case enums.RenderJobStatusFailed:
			errorMsg := "render job failed"
			_ = s.deploymentRepo.UpdateStatus(ctx, job.DeploymentID, enums.DeploymentStatusFailed, &errorMsg)
		}
	}
	
	return nil
}

// Delete deletes a render job
func (s *serviceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.renderJobRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrRenderJobNotFound) {
			return ErrRenderJobNotFound
		}
		return err
	}
	
	return nil
}
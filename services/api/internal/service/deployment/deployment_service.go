package deployment

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/deployment"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/enums"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/environment"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/release"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	deploymentRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/deployment"
	envRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/environment"
	releaseRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/release"
)

var (
	ErrDeploymentNotFound = errors.New("deployment not found")
	ErrReleaseNotFound    = errors.New("release not found")
	ErrEnvironmentNotFound = errors.New("environment not found")
	ErrInvalidStatus      = errors.New("invalid deployment status")
)

// Service defines the interface for deployment business logic
type Service interface {
	Create(ctx context.Context, req CreateRequest) (*deployment.Deployment, error)
	GetByID(ctx context.Context, id uuid.UUID) (*deployment.Deployment, error)
	GetWithRenderJob(ctx context.Context, id uuid.UUID) (*deployment.Deployment, error)
	List(ctx context.Context, filter ListFilter) ([]deployment.Deployment, int64, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*deployment.Deployment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status enums.DeploymentStatus, lastError *string) error
	Delete(ctx context.Context, id uuid.UUID) error
	CalculateIntentDigest(ctx context.Context, deploymentID uuid.UUID) (string, error)
}

// CreateRequest represents a request to create a deployment
type CreateRequest struct {
	ReleaseID  uuid.UUID              `json:"release_id"`
	EnvID      uuid.UUID              `json:"env_id"`
	ProjectID  uuid.UUID              `json:"project_id"`
	TraceID    *uuid.UUID             `json:"trace_id,omitempty"`
	CreatedBy  *string                `json:"created_by,omitempty"`
	IntentJSON map[string]interface{} `json:"intent_json,omitempty"`
}

// UpdateRequest represents a request to update a deployment
type UpdateRequest struct {
	Status         *enums.DeploymentStatus `json:"status,omitempty"`
	IntentRevision *string                 `json:"intent_revision,omitempty"`
	IntentDigest   *string                 `json:"intent_digest,omitempty"`
	IntentJSON     map[string]interface{}  `json:"intent_json,omitempty"`
	LastError      *string                 `json:"last_error,omitempty"`
}

// ListFilter contains filter parameters for listing deployments
type ListFilter struct {
	ReleaseID  *uuid.UUID
	EnvID      *uuid.UUID
	ProjectID  *uuid.UUID
	Status     *enums.DeploymentStatus
	CreatedBy  *string
	Since      *time.Time
	Until      *time.Time
	Pagination *base.Pagination
	Sort       *base.Sort
}

// serviceImpl implements Service interface
type serviceImpl struct {
	txManager      base.TxManager
	deploymentRepo deploymentRepo.Repository
	releaseRepo    releaseRepo.Repository
	envRepo        envRepo.Repository
	renderJobRepo  deploymentRepo.RenderJobRepository
}

// NewService creates a new deployment service
func NewService(
	txManager base.TxManager,
	deploymentRepo deploymentRepo.Repository,
	releaseRepo releaseRepo.Repository,
	envRepo envRepo.Repository,
	renderJobRepo deploymentRepo.RenderJobRepository,
) Service {
	return &serviceImpl{
		txManager:      txManager,
		deploymentRepo: deploymentRepo,
		releaseRepo:    releaseRepo,
		envRepo:        envRepo,
		renderJobRepo:  renderJobRepo,
	}
}

// Create creates a new deployment
func (s *serviceImpl) Create(ctx context.Context, req CreateRequest) (*deployment.Deployment, error) {
	// Validate release exists
	rel, err := s.releaseRepo.GetByID(ctx, req.ReleaseID)
	if err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			return nil, ErrReleaseNotFound
		}
		return nil, err
	}
	
	// Validate environment exists
	env, err := s.envRepo.GetByID(ctx, req.EnvID)
	if err != nil {
		if errors.Is(err, envRepo.ErrEnvironmentNotFound) {
			return nil, ErrEnvironmentNotFound
		}
		return nil, err
	}
	
	// Create deployment
	d := &deployment.Deployment{
		TraceID:   req.TraceID,
		ReleaseID: req.ReleaseID,
		EnvID:     req.EnvID,
		ProjectID: req.ProjectID,
		Status:    enums.DeploymentStatusPending,
		CreatedBy: req.CreatedBy,
	}
	
	if req.IntentJSON != nil {
		d.IntentJSON = deployment.JSONB(req.IntentJSON)
		
		// Calculate intent digest
		digest, err := s.calculateIntentDigestFromJSON(req.IntentJSON, rel, env)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate intent digest: %w", err)
		}
		d.IntentDigest = &digest
	}
	
	if err := s.deploymentRepo.Create(ctx, d); err != nil {
		return nil, err
	}
	
	return d, nil
}

// GetByID retrieves a deployment by ID
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*deployment.Deployment, error) {
	d, err := s.deploymentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrDeploymentNotFound) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}
	
	return d, nil
}

// GetWithRenderJob retrieves a deployment with its render job
func (s *serviceImpl) GetWithRenderJob(ctx context.Context, id uuid.UUID) (*deployment.Deployment, error) {
	d, err := s.deploymentRepo.GetWithRenderJob(ctx, id)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrDeploymentNotFound) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}
	
	return d, nil
}

// List retrieves deployments with filters
func (s *serviceImpl) List(ctx context.Context, filter ListFilter) ([]deployment.Deployment, int64, error) {
	repoFilter := deploymentRepo.ListFilter{
		ReleaseID:  filter.ReleaseID,
		EnvID:      filter.EnvID,
		ProjectID:  filter.ProjectID,
		Status:     filter.Status,
		CreatedBy:  filter.CreatedBy,
		Since:      filter.Since,
		Until:      filter.Until,
		Pagination: filter.Pagination,
		Sort:       filter.Sort,
	}
	
	return s.deploymentRepo.List(ctx, repoFilter)
}

// Update updates a deployment
func (s *serviceImpl) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*deployment.Deployment, error) {
	d, err := s.deploymentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrDeploymentNotFound) {
			return nil, ErrDeploymentNotFound
		}
		return nil, err
	}
	
	// Apply updates
	if req.Status != nil {
		if err := s.validateStatusTransition(d.Status, *req.Status); err != nil {
			return nil, err
		}
		d.Status = *req.Status
	}
	
	if req.IntentRevision != nil {
		d.IntentRevision = req.IntentRevision
	}
	
	if req.IntentDigest != nil {
		d.IntentDigest = req.IntentDigest
	}
	
	if req.IntentJSON != nil {
		d.IntentJSON = deployment.JSONB(req.IntentJSON)
	}
	
	if req.LastError != nil {
		d.LastError = req.LastError
	}
	
	if err := s.deploymentRepo.Update(ctx, d); err != nil {
		return nil, err
	}
	
	return d, nil
}

// UpdateStatus updates the status of a deployment
func (s *serviceImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status enums.DeploymentStatus, lastError *string) error {
	// Get current deployment to validate transition
	d, err := s.deploymentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrDeploymentNotFound) {
			return ErrDeploymentNotFound
		}
		return err
	}
	
	if err := s.validateStatusTransition(d.Status, status); err != nil {
		return err
	}
	
	return s.deploymentRepo.UpdateStatus(ctx, id, status, lastError)
}

// Delete deletes a deployment
func (s *serviceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.deploymentRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, deploymentRepo.ErrDeploymentNotFound) {
			return ErrDeploymentNotFound
		}
		return err
	}
	
	return nil
}

// CalculateIntentDigest calculates the intent digest for a deployment
func (s *serviceImpl) CalculateIntentDigest(ctx context.Context, deploymentID uuid.UUID) (string, error) {
	d, err := s.deploymentRepo.GetByID(ctx, deploymentID)
	if err != nil {
		return "", err
	}
	
	rel, err := s.releaseRepo.GetByID(ctx, d.ReleaseID)
	if err != nil {
		return "", err
	}
	
	env, err := s.envRepo.GetByID(ctx, d.EnvID)
	if err != nil {
		return "", err
	}
	
	// Create intent structure
	intent := map[string]interface{}{
		"release_id":      d.ReleaseID.String(),
		"env_id":          d.EnvID.String(),
		"release_digest":  rel.OCIDigest,
		"release_content": rel.ContentHash,
		"env_name":        env.Name,
		"env_cluster":     env.Cluster,
	}
	
	// Add render job info if exists
	renderJob, err := s.renderJobRepo.GetByDeploymentID(ctx, deploymentID)
	if err == nil && renderJob != nil {
		intent["renderer_version"] = renderJob.RendererVersion
		intent["module_versions"] = renderJob.ModuleVersions
	}
	
	// Calculate hash
	jsonBytes, err := json.Marshal(intent)
	if err != nil {
		return "", err
	}
	
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", hash), nil
}

// calculateIntentDigestFromJSON calculates intent digest from JSON
func (s *serviceImpl) calculateIntentDigestFromJSON(intentJSON map[string]interface{}, rel *release.Release, env *environment.Environment) (string, error) {
	// Combine release, environment, and intent JSON
	combined := map[string]interface{}{
		"release": map[string]interface{}{
			"id":           rel.ID.String(),
			"oci_digest":   rel.OCIDigest,
			"content_hash": rel.ContentHash,
			"values_hash":  rel.ValuesHash,
		},
		"environment": map[string]interface{}{
			"id":      env.ID.String(),
			"name":    env.Name,
			"cluster": env.Cluster,
		},
		"intent": intentJSON,
	}
	
	jsonBytes, err := json.Marshal(combined)
	if err != nil {
		return "", err
	}
	
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", hash), nil
}

// validateStatusTransition validates deployment status transitions
func (s *serviceImpl) validateStatusTransition(from, to enums.DeploymentStatus) error {
	// Define valid transitions
	validTransitions := map[enums.DeploymentStatus][]enums.DeploymentStatus{
		enums.DeploymentStatusPending: {
			enums.DeploymentStatusRendered,
			enums.DeploymentStatusFailed,
		},
		enums.DeploymentStatusRendered: {
			enums.DeploymentStatusPushed,
			enums.DeploymentStatusFailed,
		},
		enums.DeploymentStatusPushed: {
			enums.DeploymentStatusReconciling,
			enums.DeploymentStatusFailed,
		},
		enums.DeploymentStatusReconciling: {
			enums.DeploymentStatusHealthy,
			enums.DeploymentStatusDegraded,
			enums.DeploymentStatusFailed,
		},
		enums.DeploymentStatusHealthy: {
			enums.DeploymentStatusDegraded,
			enums.DeploymentStatusRolledBack,
			enums.DeploymentStatusFailed,
		},
		enums.DeploymentStatusDegraded: {
			enums.DeploymentStatusHealthy,
			enums.DeploymentStatusRolledBack,
			enums.DeploymentStatusFailed,
		},
		enums.DeploymentStatusFailed: {
			enums.DeploymentStatusPending, // Allow retry
			enums.DeploymentStatusRolledBack,
		},
		enums.DeploymentStatusRolledBack: {
			// Terminal state
		},
	}
	
	allowed, exists := validTransitions[from]
	if !exists {
		return fmt.Errorf("%w: unknown status %s", ErrInvalidStatus, from)
	}
	
	for _, validTo := range allowed {
		if validTo == to {
			return nil
		}
	}
	
	return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStatus, from, to)
}
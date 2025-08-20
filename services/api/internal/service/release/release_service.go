package release

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/enums"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/release"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
	artifactRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/artifact"
	releaseRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/release"
)

var (
	ErrReleaseSealed        = errors.New("release is sealed and cannot be modified")
	ErrInvalidArtifactField = errors.New("invalid artifact field")
	ErrArtifactNotFound     = errors.New("artifact not found")
	ErrProjectNotFound      = errors.New("project not found")
	ErrInvalidStatus        = errors.New("invalid release status")
)

// Service defines the interface for release business logic
type Service interface {
	// Release operations
	Create(ctx context.Context, req CreateRequest) (*release.Release, error)
	GetByID(ctx context.Context, id uuid.UUID) (*release.Release, error)
	GetByProjectAndKey(ctx context.Context, projectID uuid.UUID, key string) (*release.Release, error)
	List(ctx context.Context, filter ListFilter) ([]release.Release, int64, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*release.Release, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Seal(ctx context.Context, id uuid.UUID) error

	// Module operations
	ListModules(ctx context.Context, releaseID uuid.UUID) ([]release.ReleaseModule, error)
	CreateModules(ctx context.Context, releaseID uuid.UUID, modules []ModuleRequest) error
	UpdateModule(ctx context.Context, releaseID uuid.UUID, moduleKey string, req ModuleRequest) error
	DeleteModule(ctx context.Context, releaseID uuid.UUID, moduleKey string) error

	// Artifact operations
	ListArtifacts(ctx context.Context, releaseID uuid.UUID) ([]release.ReleaseArtifact, error)
	AttachArtifact(ctx context.Context, releaseID uuid.UUID, req ArtifactLinkRequest) error
	DetachArtifact(ctx context.Context, releaseID uuid.UUID, artifactID uuid.UUID, role string) error
}

// CreateRequest represents a request to create a release
type CreateRequest struct {
	ProjectID      uuid.UUID              `json:"project_id"`
	ReleaseKey     string                 `json:"release_key"`
	TraceID        *uuid.UUID             `json:"trace_id,omitempty"`
	SourceCommit   string                 `json:"source_commit"`
	SourceBranch   *string                `json:"source_branch,omitempty"`
	Tag            *string                `json:"tag,omitempty"`
	Status         *enums.ReleaseStatus   `json:"status,omitempty"`
	OCIRef         *string                `json:"oci_ref,omitempty"`
	OCIDigest      *string                `json:"oci_digest,omitempty"`
	ValuesHash     *string                `json:"values_hash,omitempty"`
	ValuesSnapshot map[string]interface{} `json:"values_snapshot,omitempty"`
	ContentHash    *string                `json:"content_hash,omitempty"`
	CreatedBy      *string                `json:"created_by,omitempty"`
	Modules        []ModuleRequest        `json:"modules,omitempty"`
	Artifacts      []ArtifactLinkRequest  `json:"artifacts,omitempty"`
}

// UpdateRequest represents a request to update a release
type UpdateRequest struct {
	Status              *enums.ReleaseStatus `json:"status,omitempty"`
	OCIRef              *string              `json:"oci_ref,omitempty"`
	OCIDigest           *string              `json:"oci_digest,omitempty"`
	Signed              *bool                `json:"signed,omitempty"`
	SigIssuer           *string              `json:"sig_issuer,omitempty"`
	SigSubject          *string              `json:"sig_subject,omitempty"`
	SignatureVerifiedAt *time.Time           `json:"signature_verified_at,omitempty"`
}

// ListFilter contains filter parameters for listing releases
type ListFilter struct {
	ProjectID  *uuid.UUID
	ReleaseKey *string
	Status     *enums.ReleaseStatus
	OCIDigest  *string
	Tag        *string
	CreatedBy  *string
	Since      *time.Time
	Until      *time.Time
	Pagination *base.Pagination
	Sort       *base.Sort
}

// ModuleRequest represents a request to create/update a module
type ModuleRequest struct {
	ModuleKey  string  `json:"module_key"`
	Name       string  `json:"name"`
	ModuleType string  `json:"module_type"`
	Version    *string `json:"version,omitempty"`
	Registry   *string `json:"registry,omitempty"`
	OCIRef     *string `json:"oci_ref,omitempty"`
	OCIDigest  *string `json:"oci_digest,omitempty"`
	GitURL     *string `json:"git_url,omitempty"`
	GitRef     *string `json:"git_ref,omitempty"`
	Path       *string `json:"path,omitempty"`
}

// ArtifactLinkRequest represents a request to link an artifact
type ArtifactLinkRequest struct {
	ArtifactID  uuid.UUID `json:"artifact_id"`
	Role        string    `json:"role"`
	ArtifactKey *string   `json:"artifact_key,omitempty"`
}

// serviceImpl implements Service interface
type serviceImpl struct {
	txManager           base.TxManager
	releaseRepo         releaseRepo.Repository
	moduleRepo          releaseRepo.ModuleRepository
	releaseArtifactRepo releaseRepo.ArtifactRepository
	artifactRepo        artifactRepo.Repository
}

// NewService creates a new release service
func NewService(
	txManager base.TxManager,
	releaseRepo releaseRepo.Repository,
	moduleRepo releaseRepo.ModuleRepository,
	releaseArtifactRepo releaseRepo.ArtifactRepository,
	artifactRepo artifactRepo.Repository,
) Service {
	return &serviceImpl{
		txManager:           txManager,
		releaseRepo:         releaseRepo,
		moduleRepo:          moduleRepo,
		releaseArtifactRepo: releaseArtifactRepo,
		artifactRepo:        artifactRepo,
	}
}

// Create creates a new release with modules and artifacts
func (s *serviceImpl) Create(ctx context.Context, req CreateRequest) (*release.Release, error) {
	var createdRelease *release.Release

	err := s.txManager.InTx(ctx, func(ctx context.Context) error {
		// Create the release
		rel := &release.Release{
			ProjectID:    req.ProjectID,
			ReleaseKey:   req.ReleaseKey,
			TraceID:      req.TraceID,
			SourceCommit: req.SourceCommit,
			SourceBranch: req.SourceBranch,
			Tag:          req.Tag,
			Status:       enums.ReleaseStatusDraft, // Default to draft
			OCIRef:       req.OCIRef,
			OCIDigest:    req.OCIDigest,
			ValuesHash:   req.ValuesHash,
			ContentHash:  req.ContentHash,
			CreatedBy:    req.CreatedBy,
		}

		if req.Status != nil {
			rel.Status = *req.Status
		}

		if req.ValuesSnapshot != nil {
			rel.ValuesSnapshot = release.JSONB(req.ValuesSnapshot)
		}

		if err := s.releaseRepo.Create(ctx, rel); err != nil {
			return err
		}

		// Create modules if provided
		if len(req.Modules) > 0 {
			modules := make([]release.ReleaseModule, len(req.Modules))
			for i, m := range req.Modules {
				modules[i] = release.ReleaseModule{
					ReleaseID:  rel.ID,
					ModuleKey:  m.ModuleKey,
					Name:       m.Name,
					ModuleType: release.ModuleType(m.ModuleType),
					Version:    m.Version,
					Registry:   m.Registry,
					OCIRef:     m.OCIRef,
					OCIDigest:  m.OCIDigest,
					GitURL:     m.GitURL,
					GitRef:     m.GitRef,
					Path:       m.Path,
				}
			}
			if err := s.moduleRepo.CreateBulk(ctx, modules); err != nil {
				return err
			}
		}

		// Link artifacts if provided
		if len(req.Artifacts) > 0 {
			artifacts := make([]release.ReleaseArtifact, len(req.Artifacts))
			for i, a := range req.Artifacts {
				// Verify artifact exists
				exists, err := s.artifactRepo.ExistsByID(ctx, a.ArtifactID)
				if err != nil {
					return err
				}
				if !exists {
					return fmt.Errorf("%w: %s", ErrArtifactNotFound, a.ArtifactID)
				}

				artifacts[i] = release.ReleaseArtifact{
					ReleaseID:   rel.ID,
					ArtifactID:  a.ArtifactID,
					Role:        a.Role,
					ArtifactKey: a.ArtifactKey,
				}
			}
			if err := s.releaseArtifactRepo.CreateBulk(ctx, artifacts); err != nil {
				return err
			}
		}

		createdRelease = rel
		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdRelease, nil
}

// GetByID retrieves a release by ID
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*release.Release, error) {
	return s.releaseRepo.GetByID(ctx, id)
}

// GetByProjectAndKey retrieves a release by project ID and key
func (s *serviceImpl) GetByProjectAndKey(ctx context.Context, projectID uuid.UUID, key string) (*release.Release, error) {
	return s.releaseRepo.GetByProjectAndKey(ctx, projectID, key)
}

// List retrieves releases with filters
func (s *serviceImpl) List(ctx context.Context, filter ListFilter) ([]release.Release, int64, error) {
	repoFilter := releaseRepo.ListFilter{
		ProjectID:  filter.ProjectID,
		ReleaseKey: filter.ReleaseKey,
		Status:     filter.Status,
		OCIDigest:  filter.OCIDigest,
		Tag:        filter.Tag,
		CreatedBy:  filter.CreatedBy,
		Since:      filter.Since,
		Until:      filter.Until,
		Pagination: filter.Pagination,
		Sort:       filter.Sort,
	}

	return s.releaseRepo.List(ctx, repoFilter)
}

// Update updates a release
func (s *serviceImpl) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*release.Release, error) {
	// Check if release is sealed
	sealed, err := s.releaseRepo.IsSealed(ctx, id)
	if err != nil {
		return nil, err
	}

	if sealed {
		// Only allow certain updates on sealed releases
		if req.Status != nil || req.OCIRef != nil || req.OCIDigest != nil {
			return nil, ErrReleaseSealed
		}
	}

	// Get existing release
	rel, err := s.releaseRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Status != nil {
		rel.Status = *req.Status
	}
	if req.OCIRef != nil {
		rel.OCIRef = req.OCIRef
	}
	if req.OCIDigest != nil {
		rel.OCIDigest = req.OCIDigest
	}
	if req.Signed != nil {
		rel.Signed = *req.Signed
	}
	if req.SigIssuer != nil {
		rel.SigIssuer = req.SigIssuer
	}
	if req.SigSubject != nil {
		rel.SigSubject = req.SigSubject
	}
	if req.SignatureVerifiedAt != nil {
		rel.SignatureVerifiedAt = req.SignatureVerifiedAt
	}

	if err := s.releaseRepo.Update(ctx, rel); err != nil {
		return nil, err
	}

	return rel, nil
}

// Delete deletes a release and all its sub-resources
func (s *serviceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	// If sealed, do not allow deletion
	sealed, err := s.releaseRepo.IsSealed(ctx, id)
	if err != nil {
		return err
	}
	if sealed {
		return ErrReleaseSealed
	}
	// The repository handles cascade deletion and checks for deployment references
	return s.releaseRepo.Delete(ctx, id)
}

// Seal seals a release, making it immutable
func (s *serviceImpl) Seal(ctx context.Context, id uuid.UUID) error {
	rel, err := s.releaseRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if rel.Status == enums.ReleaseStatusSealed {
		return nil // Already sealed
	}

	rel.Status = enums.ReleaseStatusSealed
	return s.releaseRepo.Update(ctx, rel)
}

// ListModules lists all modules for a release
func (s *serviceImpl) ListModules(ctx context.Context, releaseID uuid.UUID) ([]release.ReleaseModule, error) {
	return s.moduleRepo.ListByRelease(ctx, releaseID)
}

// CreateModules creates modules for a release
func (s *serviceImpl) CreateModules(ctx context.Context, releaseID uuid.UUID, modules []ModuleRequest) error {
	// Check if release is sealed
	sealed, err := s.releaseRepo.IsSealed(ctx, releaseID)
	if err != nil {
		return err
	}
	if sealed {
		return ErrReleaseSealed
	}

	releaseModules := make([]release.ReleaseModule, len(modules))
	for i, m := range modules {
		releaseModules[i] = release.ReleaseModule{
			ReleaseID:  releaseID,
			ModuleKey:  m.ModuleKey,
			Name:       m.Name,
			ModuleType: release.ModuleType(m.ModuleType),
			Version:    m.Version,
			Registry:   m.Registry,
			OCIRef:     m.OCIRef,
			OCIDigest:  m.OCIDigest,
			GitURL:     m.GitURL,
			GitRef:     m.GitRef,
			Path:       m.Path,
		}
	}

	return s.moduleRepo.CreateBulk(ctx, releaseModules)
}

// UpdateModule updates a module
func (s *serviceImpl) UpdateModule(ctx context.Context, releaseID uuid.UUID, moduleKey string, req ModuleRequest) error {
	// Check if release is sealed
	sealed, err := s.releaseRepo.IsSealed(ctx, releaseID)
	if err != nil {
		return err
	}
	if sealed {
		return ErrReleaseSealed
	}

	module, err := s.moduleRepo.GetByReleaseAndKey(ctx, releaseID, moduleKey)
	if err != nil {
		return err
	}

	// Update fields
	module.Name = req.Name
	module.ModuleType = release.ModuleType(req.ModuleType)
	module.Version = req.Version
	module.Registry = req.Registry
	module.OCIRef = req.OCIRef
	module.OCIDigest = req.OCIDigest
	module.GitURL = req.GitURL
	module.GitRef = req.GitRef
	module.Path = req.Path

	return s.moduleRepo.Update(ctx, module)
}

// DeleteModule deletes a module
func (s *serviceImpl) DeleteModule(ctx context.Context, releaseID uuid.UUID, moduleKey string) error {
	// Check if release is sealed
	sealed, err := s.releaseRepo.IsSealed(ctx, releaseID)
	if err != nil {
		return err
	}
	if sealed {
		return ErrReleaseSealed
	}

	return s.moduleRepo.Delete(ctx, releaseID, moduleKey)
}

// ListArtifacts lists all artifacts for a release
func (s *serviceImpl) ListArtifacts(ctx context.Context, releaseID uuid.UUID) ([]release.ReleaseArtifact, error) {
	return s.releaseArtifactRepo.ListByRelease(ctx, releaseID)
}

// AttachArtifact attaches an artifact to a release
func (s *serviceImpl) AttachArtifact(ctx context.Context, releaseID uuid.UUID, req ArtifactLinkRequest) error {
	// Check if release is sealed
	sealed, err := s.releaseRepo.IsSealed(ctx, releaseID)
	if err != nil {
		return err
	}
	if sealed {
		return ErrReleaseSealed
	}

	// Verify artifact exists
	exists, err := s.artifactRepo.ExistsByID(ctx, req.ArtifactID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", ErrArtifactNotFound, req.ArtifactID)
	}

	releaseArtifact := &release.ReleaseArtifact{
		ReleaseID:   releaseID,
		ArtifactID:  req.ArtifactID,
		Role:        req.Role,
		ArtifactKey: req.ArtifactKey,
	}

	return s.releaseArtifactRepo.Create(ctx, releaseArtifact)
}

// DetachArtifact detaches an artifact from a release
func (s *serviceImpl) DetachArtifact(ctx context.Context, releaseID uuid.UUID, artifactID uuid.UUID, role string) error {
	// Check if release is sealed
	sealed, err := s.releaseRepo.IsSealed(ctx, releaseID)
	if err != nil {
		return err
	}
	if sealed {
		return ErrReleaseSealed
	}

	return s.releaseArtifactRepo.Delete(ctx, releaseID, artifactID, role)
}

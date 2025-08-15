package artifact

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/artifact"
	artifactRepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/artifact"
	base "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
	buildRepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/build"
)

var (
	ErrArtifactNotFound = errors.New("artifact not found")
	ErrArtifactExists   = errors.New("artifact with this digest already exists")
	ErrBuildNotFound    = errors.New("build not found")
	ErrArtifactInUse    = errors.New("artifact is referenced by releases and cannot be deleted")
)

// Service defines the interface for artifact business logic
type Service interface {
	Create(ctx context.Context, req CreateRequest) (*artifact.Artifact, error)
	GetByID(ctx context.Context, id uuid.UUID) (*artifact.Artifact, error)
	GetByDigest(ctx context.Context, digest string) (*artifact.Artifact, error)
	List(ctx context.Context, filter ListFilter) ([]artifact.Artifact, int64, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*artifact.Artifact, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// CreateRequest represents a request to create an artifact
type CreateRequest struct {
	BuildID   uuid.UUID              `json:"build_id"`
	Kind      string                 `json:"kind"`
	Name      *string                `json:"name,omitempty"`
	URI       *string                `json:"uri,omitempty"`
	MediaType *string                `json:"media_type,omitempty"`
	Digest    *string                `json:"digest,omitempty"`
	SizeBytes *int64                 `json:"size_bytes,omitempty"`
	Labels    map[string]interface{} `json:"labels,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateRequest represents a request to update an artifact
type UpdateRequest struct {
	Labels   map[string]interface{} `json:"labels,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ListFilter contains filter parameters for listing artifacts
type ListFilter struct {
	BuildID    *uuid.UUID
	Kind       *string
	Name       *string
	Digest     *string
	MediaType  *string
	Pagination *base.Pagination
	Sort       *base.Sort
}

// serviceImpl implements Service interface
type serviceImpl struct {
	txManager    base.TxManager
	artifactRepo artifactRepo.Repository
	buildRepo    buildRepo.Repository
}

// NewService creates a new artifact service
func NewService(
	txManager base.TxManager,
	artifactRepo artifactRepo.Repository,
	buildRepo buildRepo.Repository,
) Service {
	return &serviceImpl{
		txManager:    txManager,
		artifactRepo: artifactRepo,
		buildRepo:    buildRepo,
	}
}

// Create creates a new artifact
func (s *serviceImpl) Create(ctx context.Context, req CreateRequest) (*artifact.Artifact, error) {
	// Verify build exists
	_, err := s.buildRepo.GetByID(ctx, req.BuildID)
	if err != nil {
		if errors.Is(err, buildRepo.ErrBuildNotFound) {
			return nil, ErrBuildNotFound
		}
		return nil, err
	}
	
	// Create artifact
	a := &artifact.Artifact{
		BuildID:   req.BuildID,
		Kind:      req.Kind,
		Name:      req.Name,
		URI:       req.URI,
		MediaType: req.MediaType,
		Digest:    req.Digest,
		SizeBytes: req.SizeBytes,
	}
	
	if req.Labels != nil {
		a.Labels = artifact.JSONB(req.Labels)
	}
	
	if req.Metadata != nil {
		a.Metadata = artifact.JSONB(req.Metadata)
	}
	
	if err := s.artifactRepo.Create(ctx, a); err != nil {
		if errors.Is(err, artifactRepo.ErrArtifactExists) {
			return nil, ErrArtifactExists
		}
		return nil, err
	}
	
	return a, nil
}

// GetByID retrieves an artifact by ID
func (s *serviceImpl) GetByID(ctx context.Context, id uuid.UUID) (*artifact.Artifact, error) {
	a, err := s.artifactRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, artifactRepo.ErrArtifactNotFound) {
			return nil, ErrArtifactNotFound
		}
		return nil, err
	}
	
	return a, nil
}

// GetByDigest retrieves an artifact by digest
func (s *serviceImpl) GetByDigest(ctx context.Context, digest string) (*artifact.Artifact, error) {
	a, err := s.artifactRepo.GetByDigest(ctx, digest)
	if err != nil {
		if errors.Is(err, artifactRepo.ErrArtifactNotFound) {
			return nil, ErrArtifactNotFound
		}
		return nil, err
	}
	
	return a, nil
}

// List retrieves artifacts with filters
func (s *serviceImpl) List(ctx context.Context, filter ListFilter) ([]artifact.Artifact, int64, error) {
	repoFilter := artifactRepo.ListFilter{
		BuildID:    filter.BuildID,
		Kind:       filter.Kind,
		Name:       filter.Name,
		Digest:     filter.Digest,
		MediaType:  filter.MediaType,
		Pagination: filter.Pagination,
		Sort:       filter.Sort,
	}
	
	return s.artifactRepo.List(ctx, repoFilter)
}

// Update updates an artifact (only labels and metadata)
func (s *serviceImpl) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (*artifact.Artifact, error) {
	a, err := s.artifactRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, artifactRepo.ErrArtifactNotFound) {
			return nil, ErrArtifactNotFound
		}
		return nil, err
	}
	
	// Only update labels and metadata
	if req.Labels != nil {
		a.Labels = artifact.JSONB(req.Labels)
	}
	
	if req.Metadata != nil {
		a.Metadata = artifact.JSONB(req.Metadata)
	}
	
	if err := s.artifactRepo.Update(ctx, a); err != nil {
		return nil, err
	}
	
	return a, nil
}

// Delete deletes an artifact (checks for references)
func (s *serviceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.artifactRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, artifactRepo.ErrArtifactNotFound) {
			return ErrArtifactNotFound
		}
		// Check if it's a reference constraint error
		if errors.Is(err, errors.New("artifact is referenced by releases and cannot be deleted")) {
			return ErrArtifactInUse
		}
		return err
	}
	
	return nil
}
package artifact

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/artifact"
	base "github.com/input-output-hk/catalyst-forge/services/api/internal/repository"
)

var (
	ErrArtifactNotFound = errors.New("artifact not found")
	ErrArtifactExists   = errors.New("artifact with this digest already exists")
	ErrArtifactInUse    = errors.New("artifact is referenced by releases and cannot be deleted")
)

// Repository defines the interface for artifact operations
type Repository interface {
	Create(ctx context.Context, artifact *artifact.Artifact) error
	GetByID(ctx context.Context, id uuid.UUID) (*artifact.Artifact, error)
	GetByDigest(ctx context.Context, digest string) (*artifact.Artifact, error)
	List(ctx context.Context, filter ListFilter) ([]artifact.Artifact, int64, error)
	Update(ctx context.Context, artifact *artifact.Artifact) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
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

// repositoryImpl implements Repository interface
type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

// Create creates a new artifact
func (r *repositoryImpl) Create(ctx context.Context, a *artifact.Artifact) error {
	db := base.GetDB(ctx, r.db)

	if err := db.Create(a).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrArtifactExists
		}
		return err
	}

	return nil
}

// GetByID retrieves an artifact by ID
func (r *repositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*artifact.Artifact, error) {
	db := base.GetDB(ctx, r.db)

	var a artifact.Artifact
	if err := db.Where("id = ?", id).First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArtifactNotFound
		}
		return nil, err
	}

	return &a, nil
}

// GetByDigest retrieves an artifact by digest
func (r *repositoryImpl) GetByDigest(ctx context.Context, digest string) (*artifact.Artifact, error) {
	db := base.GetDB(ctx, r.db)

	var a artifact.Artifact
	if err := db.Where("digest = ?", digest).First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArtifactNotFound
		}
		return nil, err
	}

	return &a, nil
}

// List retrieves artifacts with filters and pagination
func (r *repositoryImpl) List(ctx context.Context, filter ListFilter) ([]artifact.Artifact, int64, error) {
	db := base.GetDB(ctx, r.db)

	query := db.Model(&artifact.Artifact{})

	// Apply filters
	if filter.BuildID != nil {
		query = query.Where("build_id = ?", *filter.BuildID)
	}
	if filter.Kind != nil {
		query = query.Where("kind = ?", *filter.Kind)
	}
	if filter.Name != nil {
		query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
	}
	if filter.Digest != nil {
		query = query.Where("digest = ?", *filter.Digest)
	}
	if filter.MediaType != nil {
		query = query.Where("media_type = ?", *filter.MediaType)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	query = base.ApplySort(query, filter.Sort, "created_at DESC")

	// Apply pagination
	query = base.ApplyPagination(query, filter.Pagination)

	// Fetch results
	var artifacts []artifact.Artifact
	if err := query.Find(&artifacts).Error; err != nil {
		return nil, 0, err
	}

	return artifacts, total, nil
}

// Update updates an artifact (mainly for labels/metadata)
func (r *repositoryImpl) Update(ctx context.Context, a *artifact.Artifact) error {
	db := base.GetDB(ctx, r.db)

	// Only allow updating certain fields
	updates := map[string]interface{}{
		"labels":   a.Labels,
		"metadata": a.Metadata,
	}

	result := db.Model(&artifact.Artifact{}).Where("id = ?", a.ID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrArtifactNotFound
	}

	return nil
}

// Delete deletes an artifact by ID
func (r *repositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	db := base.GetDB(ctx, r.db)

	// Check if artifact is referenced by any releases
	var count int64
	db.Table("release_artifact").Where("artifact_id = ?", id).Count(&count)
	if count > 0 {
		return ErrArtifactInUse
	}

	result := db.Where("id = ?", id).Delete(&artifact.Artifact{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrArtifactNotFound
	}

	return nil
}

// ExistsByID checks if an artifact exists by ID
func (r *repositoryImpl) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	db := base.GetDB(ctx, r.db)

	var count int64
	if err := db.Model(&artifact.Artifact{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

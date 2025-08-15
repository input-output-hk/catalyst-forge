package release

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/release"
	base "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
)

var (
	ErrReleaseArtifactNotFound = errors.New("release artifact not found")
	ErrReleaseArtifactExists   = errors.New("release artifact already exists")
)

// ArtifactRepository defines the interface for release artifact operations
type ArtifactRepository interface {
	Create(ctx context.Context, artifact *release.ReleaseArtifact) error
	CreateBulk(ctx context.Context, artifacts []release.ReleaseArtifact) error
	GetByReleaseAndArtifact(ctx context.Context, releaseID, artifactID uuid.UUID, role string) (*release.ReleaseArtifact, error)
	GetByReleaseAndKey(ctx context.Context, releaseID uuid.UUID, artifactKey string) (*release.ReleaseArtifact, error)
	ListByRelease(ctx context.Context, releaseID uuid.UUID) ([]release.ReleaseArtifact, error)
	Delete(ctx context.Context, releaseID, artifactID uuid.UUID, role string) error
	DeleteByRelease(ctx context.Context, releaseID uuid.UUID) error
}

// artifactRepositoryImpl implements ArtifactRepository interface
type artifactRepositoryImpl struct {
	db *gorm.DB
}

// NewArtifactRepository creates a new artifact repository instance
func NewArtifactRepository(db *gorm.DB) ArtifactRepository {
	return &artifactRepositoryImpl{db: db}
}

// Create creates a new release artifact link
func (r *artifactRepositoryImpl) Create(ctx context.Context, artifact *release.ReleaseArtifact) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(artifact).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrReleaseArtifactExists
		}
		return err
	}
	
	return nil
}

// CreateBulk creates multiple release artifact links
func (r *artifactRepositoryImpl) CreateBulk(ctx context.Context, artifacts []release.ReleaseArtifact) error {
	if len(artifacts) == 0 {
		return nil
	}
	
	db := base.GetDB(ctx, r.db)
	
	if err := db.Create(&artifacts).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrReleaseArtifactExists
		}
		return err
	}
	
	return nil
}

// GetByReleaseAndArtifact retrieves a release artifact by release ID, artifact ID, and role
func (r *artifactRepositoryImpl) GetByReleaseAndArtifact(ctx context.Context, releaseID, artifactID uuid.UUID, role string) (*release.ReleaseArtifact, error) {
	db := base.GetDB(ctx, r.db)
	
	var artifact release.ReleaseArtifact
	if err := db.Where("release_id = ? AND artifact_id = ? AND role = ?", releaseID, artifactID, role).First(&artifact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReleaseArtifactNotFound
		}
		return nil, err
	}
	
	return &artifact, nil
}

// GetByReleaseAndKey retrieves a release artifact by release ID and artifact key
func (r *artifactRepositoryImpl) GetByReleaseAndKey(ctx context.Context, releaseID uuid.UUID, artifactKey string) (*release.ReleaseArtifact, error) {
	db := base.GetDB(ctx, r.db)
	
	var artifact release.ReleaseArtifact
	if err := db.Where("release_id = ? AND artifact_key = ?", releaseID, artifactKey).First(&artifact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReleaseArtifactNotFound
		}
		return nil, err
	}
	
	return &artifact, nil
}

// ListByRelease retrieves all artifacts for a release
func (r *artifactRepositoryImpl) ListByRelease(ctx context.Context, releaseID uuid.UUID) ([]release.ReleaseArtifact, error) {
	db := base.GetDB(ctx, r.db)
	
	var artifacts []release.ReleaseArtifact
	if err := db.Where("release_id = ?", releaseID).Order("role, artifact_key").Find(&artifacts).Error; err != nil {
		return nil, err
	}
	
	return artifacts, nil
}

// Delete deletes a release artifact link
func (r *artifactRepositoryImpl) Delete(ctx context.Context, releaseID, artifactID uuid.UUID, role string) error {
	db := base.GetDB(ctx, r.db)
	
	result := db.Where("release_id = ? AND artifact_id = ? AND role = ?", releaseID, artifactID, role).Delete(&release.ReleaseArtifact{})
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return ErrReleaseArtifactNotFound
	}
	
	return nil
}

// DeleteByRelease deletes all artifact links for a release
func (r *artifactRepositoryImpl) DeleteByRelease(ctx context.Context, releaseID uuid.UUID) error {
	db := base.GetDB(ctx, r.db)
	
	if err := db.Where("release_id = ?", releaseID).Delete(&release.ReleaseArtifact{}).Error; err != nil {
		return err
	}
	
	return nil
}
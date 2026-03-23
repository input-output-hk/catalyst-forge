package buildrepo

import (
	"time"

	build "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/build"
	"gorm.io/gorm"
)

type BuildSessionRepository interface {
	Create(session *build.BuildSession) error
	CountActive(ownerType string, ownerID string) (int64, error)
}

type buildSessionRepository struct{ db *gorm.DB }

func NewBuildSessionRepository(db *gorm.DB) BuildSessionRepository {
	return &buildSessionRepository{db: db}
}

func (r *buildSessionRepository) Create(session *build.BuildSession) error {
	return r.db.Create(session).Error
}

func (r *buildSessionRepository) CountActive(ownerType string, ownerID string) (int64, error) {
	var n int64
	now := time.Now()
	err := r.db.Model(&build.BuildSession{}).
		Where("owner_type = ? AND owner_id = ? AND expires_at > ?", ownerType, ownerID, now).
		Count(&n).Error
	return n, err
}

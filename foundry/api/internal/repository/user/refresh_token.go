package user

import (
	"time"

	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

type RefreshTokenRepository interface {
	Create(token *dbmodel.RefreshToken) error
	GetByHash(hash string) (*dbmodel.RefreshToken, error)
	MarkReplaced(oldID uint, newID uint) error
	RevokeChain(startID uint) error
	TouchUsage(id uint, at time.Time) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(token *dbmodel.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *refreshTokenRepository) GetByHash(hash string) (*dbmodel.RefreshToken, error) {
	var t dbmodel.RefreshToken
	tx := r.db.First(&t, "token_hash = ?", hash)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &t, nil
}

func (r *refreshTokenRepository) MarkReplaced(oldID uint, newID uint) error {
	return r.db.Model(&dbmodel.RefreshToken{}).Where("id = ?", oldID).Update("replaced_by", newID).Error
}

func (r *refreshTokenRepository) RevokeChain(startID uint) error {
	now := time.Now()
	return r.db.Model(&dbmodel.RefreshToken{}).
		Where("id = ? OR replaced_by = ?", startID, startID).
		Update("revoked_at", &now).Error
}

func (r *refreshTokenRepository) TouchUsage(id uint, at time.Time) error {
	return r.db.Model(&dbmodel.RefreshToken{}).Where("id = ?", id).Update("last_used_at", at).Error
}

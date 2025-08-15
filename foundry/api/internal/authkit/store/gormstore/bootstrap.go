package gormstore

import (
	"context"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
	"gorm.io/gorm"
)

// BootstrapTokenStore implements store.BootstrapStore using GORM.
type BootstrapTokenStore struct{ db *gorm.DB }

func NewBootstrapTokenStore(db *gorm.DB) *BootstrapTokenStore { return &BootstrapTokenStore{db: db} }

func (s *BootstrapTokenStore) IsTokenUsed(ctx context.Context, tokenHashHex string) (bool, error) {
	var rec BootstrapTokenUsed
	err := s.db.WithContext(ctx).Where("token_hash_hex = ?", tokenHashHex).First(&rec).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	return err == nil, err
}

func (s *BootstrapTokenStore) MarkUsed(ctx context.Context, tokenHashHex string, email string) error {
	rec := BootstrapTokenUsed{TokenHashHex: tokenHashHex, UsedByEmail: email, UsedAt: time.Now().UTC()}
	return s.db.WithContext(ctx).Create(&rec).Error
}

var _ store.BootstrapStore = (*BootstrapTokenStore)(nil)

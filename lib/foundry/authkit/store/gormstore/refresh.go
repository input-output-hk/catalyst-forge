package gormstore

import (
	"context"
	"errors"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshStore is a GORM-based implementation of store.RefreshStore.
type RefreshStore struct {
	db *gorm.DB
}

// NewRefreshStore creates a new GORM-based refresh token store.
func NewRefreshStore(db *gorm.DB) *RefreshStore {
	return &RefreshStore{
		db: db,
	}
}

// CreateFamily creates a new refresh token family for initial login.
func (s *RefreshStore) CreateFamily(ctx context.Context, userID uuid.UUID, sessionVersion int64, tokenHash []byte, expiresAt time.Time) (familyID uuid.UUID, tokenID uuid.UUID, err error) {
	familyID = uuid.New()
	tokenID = uuid.New()
	now := time.Now()

	token := &RefreshToken{
		ID:             tokenID,
		FamilyID:       familyID,
		UserID:         userID,
		Hash:           tokenHash,
		SessionVersion: sessionVersion,
		CreatedAt:      now,
		ExpiresAt:      expiresAt,
	}

	if err := s.db.WithContext(ctx).Create(token).Error; err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return familyID, tokenID, nil
}

// Rotate rotates a refresh token to a new one within the same family.
//
// This marks the previous token as rotated and creates a new one.
func (s *RefreshStore) Rotate(ctx context.Context, prevTokenID uuid.UUID, newHash []byte, now time.Time, expiresAt time.Time) (newTokenID uuid.UUID, familyID uuid.UUID, err error) {
	var prevToken RefreshToken
	
	// Start a transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Get and lock the previous token
	if err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("id = ?", prevTokenID).
		First(&prevToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, uuid.Nil, errors.New("token not found")
		}
		return uuid.Nil, uuid.Nil, err
	}

	// Check if already rotated or revoked
	if prevToken.RotatedAt != nil {
		return uuid.Nil, uuid.Nil, errors.New("token already rotated")
	}
	if prevToken.RevokedAt != nil {
		return uuid.Nil, uuid.Nil, errors.New("token revoked")
	}

	// Mark previous token as rotated
	prevToken.RotatedAt = &now
	if err := tx.Save(&prevToken).Error; err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	// Create new token
	newTokenID = uuid.New()
	newToken := &RefreshToken{
		ID:             newTokenID,
		FamilyID:       prevToken.FamilyID,
		UserID:         prevToken.UserID,
		Hash:           newHash,
		SessionVersion: prevToken.SessionVersion,
		CreatedAt:      now,
		ExpiresAt:      expiresAt,
	}

	if err := tx.Create(newToken).Error; err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return newTokenID, prevToken.FamilyID, nil
}

// GetByID retrieves a refresh token by its ID.
func (s *RefreshStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error) {
	var token RefreshToken
	
	if err := s.db.WithContext(ctx).
		Where("id = ?", id).
		First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("token not found")
		}
		return nil, err
	}

	return s.toDomain(&token), nil
}

// GetByHash retrieves a refresh token by its hash.
func (s *RefreshStore) GetByHash(ctx context.Context, hash []byte) (*domain.RefreshToken, error) {
	var token RefreshToken
	
	if err := s.db.WithContext(ctx).
		Where("hash = ?", hash).
		First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("token not found")
		}
		return nil, err
	}

	return s.toDomain(&token), nil
}

// RevokeToken revokes a specific refresh token.
func (s *RefreshStore) RevokeToken(ctx context.Context, id uuid.UUID, reason string, at time.Time) error {
	result := s.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"revoked_at": at,
			"reason":     reason,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("token not found")
	}

	return nil
}

// RevokeFamily revokes all tokens in a family.
//
// This is used for replay detection.
func (s *RefreshStore) RevokeFamily(ctx context.Context, familyID uuid.UUID, reason string, at time.Time) error {
	result := s.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Updates(map[string]interface{}{
			"revoked_at": at,
			"reason":     reason,
		})

	return result.Error
}

// RevokeUserTokens revokes all refresh tokens for a user.
func (s *RefreshStore) RevokeUserTokens(ctx context.Context, userID uuid.UUID, reason string, at time.Time) error {
	result := s.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]interface{}{
			"revoked_at": at,
			"reason":     reason,
		})

	return result.Error
}

// toDomain converts a database model to a domain entity.
func (s *RefreshStore) toDomain(token *RefreshToken) *domain.RefreshToken {
	return &domain.RefreshToken{
		ID:             token.ID,
		FamilyID:       token.FamilyID,
		UserID:         token.UserID,
		Hash:           token.Hash,
		SessionVersion: token.SessionVersion,
		CreatedAt:      token.CreatedAt,
		ExpiresAt:      token.ExpiresAt,
		RotatedAt:      token.RotatedAt,
		RevokedAt:      token.RevokedAt,
		Reason:         token.Reason,
	}
}
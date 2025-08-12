package user

import (
    "errors"
    "time"

    "github.com/google/uuid"
    dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
    "gorm.io/gorm"
)

type RefreshTokenRepository interface {
	Create(token *dbmodel.RefreshToken) error
	GetByID(id uuid.UUID) (*dbmodel.RefreshToken, error)
	GetBySecretHash(hash string) (*dbmodel.RefreshToken, error)
	CreateRotatedToken(parentID uuid.UUID, familyID uuid.UUID, userID uint, deviceID uuid.UUID, secretHash string, expiresAt time.Time, ip, userAgent *string) (*dbmodel.RefreshToken, error)
	RotateTokenAtomic(oldToken *dbmodel.RefreshToken, newToken *dbmodel.RefreshToken) error
	RevokeTokenFamily(familyID uuid.UUID) error
	RevokeToken(id uuid.UUID) error
	GetActiveTokensByDevice(deviceID uuid.UUID) ([]dbmodel.RefreshToken, error)
	GetActiveTokensByUser(userID uint) ([]dbmodel.RefreshToken, error)
	IsTokenReplaced(id uuid.UUID) (bool, error)
	GetTokenFamily(familyID uuid.UUID) ([]dbmodel.RefreshToken, error)
	GetTokensByFamily(familyID uuid.UUID) ([]dbmodel.RefreshToken, error)
	GetNewerTokensInFamily(familyID uuid.UUID, afterTime time.Time) ([]dbmodel.RefreshToken, error)
	Update(token *dbmodel.RefreshToken) error
	CleanupExpiredTokens() error
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

func (r *refreshTokenRepository) GetByID(id uuid.UUID) (*dbmodel.RefreshToken, error) {
	var token dbmodel.RefreshToken
	if err := r.db.Where("id = ? AND revoked_at IS NULL", id).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *refreshTokenRepository) GetBySecretHash(hash string) (*dbmodel.RefreshToken, error) {
	var token dbmodel.RefreshToken
	if err := r.db.Where("secret_hash = ? AND revoked_at IS NULL AND expires_at > NOW()", hash).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *refreshTokenRepository) CreateRotatedToken(parentID uuid.UUID, familyID uuid.UUID, userID uint, deviceID uuid.UUID, secretHash string, expiresAt time.Time, ip, userAgent *string) (*dbmodel.RefreshToken, error) {
	newToken := &dbmodel.RefreshToken{
		UserID:     userID,
		DeviceID:   deviceID,
		FamilyID:   familyID,
		ParentID:   &parentID,
		SecretHash: secretHash,
		ExpiresAt:  expiresAt,
		IP:         ip,
		UserAgent:  userAgent,
	}

	if err := r.db.Create(newToken).Error; err != nil {
		return nil, err
	}

	return newToken, nil
}

// RotateTokenAtomic atomically marks the old token as rotated and creates a new token
// This prevents race conditions where two parallel refresh requests could both succeed.
func (r *refreshTokenRepository) RotateTokenAtomic(oldToken *dbmodel.RefreshToken, newToken *dbmodel.RefreshToken) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// First, check if the token has already been rotated
		var existing dbmodel.RefreshToken
		if err := tx.Where("id = ? AND rotated_at IS NULL AND revoked_at IS NULL", oldToken.ID).
			First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Token already rotated or doesn't exist
				return gorm.ErrRecordNotFound
			}
			return err
		}

		// Mark the old token as rotated atomically
		now := time.Now()
		result := tx.Model(&dbmodel.RefreshToken{}).
			Where("id = ? AND rotated_at IS NULL AND revoked_at IS NULL", oldToken.ID).
			Update("rotated_at", &now)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			// Token was already rotated by another request
			return gorm.ErrRecordNotFound
		}

		// Create the new refresh token
		if err := tx.Create(newToken).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *refreshTokenRepository) RevokeTokenFamily(familyID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&dbmodel.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", &now).Error
}

func (r *refreshTokenRepository) RevokeToken(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&dbmodel.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked_at", &now).Error
}

func (r *refreshTokenRepository) GetActiveTokensByDevice(deviceID uuid.UUID) ([]dbmodel.RefreshToken, error) {
	var tokens []dbmodel.RefreshToken
	if err := r.db.Where("device_id = ? AND revoked_at IS NULL AND expires_at > NOW()", deviceID).Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *refreshTokenRepository) GetActiveTokensByUser(userID uint) ([]dbmodel.RefreshToken, error) {
	var tokens []dbmodel.RefreshToken
	if err := r.db.Where("user_id = ? AND revoked_at IS NULL AND expires_at > NOW()", userID).Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *refreshTokenRepository) IsTokenReplaced(id uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.Model(&dbmodel.RefreshToken{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *refreshTokenRepository) GetTokenFamily(familyID uuid.UUID) ([]dbmodel.RefreshToken, error) {
	var tokens []dbmodel.RefreshToken
	if err := r.db.Where("family_id = ?", familyID).Order("created_at ASC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *refreshTokenRepository) GetTokensByFamily(familyID uuid.UUID) ([]dbmodel.RefreshToken, error) {
	// Alias for GetTokenFamily for convenience
	return r.GetTokenFamily(familyID)
}

func (r *refreshTokenRepository) GetNewerTokensInFamily(familyID uuid.UUID, afterTime time.Time) ([]dbmodel.RefreshToken, error) {
	var tokens []dbmodel.RefreshToken
	if err := r.db.Where("family_id = ? AND created_at > ?", familyID, afterTime).
		Order("created_at ASC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *refreshTokenRepository) Update(token *dbmodel.RefreshToken) error {
	return r.db.Save(token).Error
}

func (r *refreshTokenRepository) CleanupExpiredTokens() error {
	return r.db.Delete(&dbmodel.RefreshToken{}, "expires_at < NOW()").Error
}

package user

import (
	"errors"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"gorm.io/gorm"
)

// BootstrapTokenRepository defines the interface for bootstrap token operations.
type BootstrapTokenRepository interface {
	// Create stores a new bootstrap token usage record
	Create(token *user.BootstrapToken) error
	// GetByTokenHash retrieves a bootstrap token by its hash
	GetByTokenHash(tokenHash string) (*user.BootstrapToken, error)
	// IsTokenUsed checks if a token hash has been used
	IsTokenUsed(tokenHash string) (bool, error)
}

// bootstrapTokenRepository implements BootstrapTokenRepository.
type bootstrapTokenRepository struct {
	db *gorm.DB
}

// NewBootstrapTokenRepository creates a new instance of BootstrapTokenRepository.
func NewBootstrapTokenRepository(db *gorm.DB) BootstrapTokenRepository {
	return &bootstrapTokenRepository{db: db}
}

// Create stores a new bootstrap token usage record.
func (r *bootstrapTokenRepository) Create(token *user.BootstrapToken) error {
	if token == nil {
		return errors.New("bootstrap token cannot be nil")
	}
	return r.db.Create(token).Error
}

// GetByTokenHash retrieves a bootstrap token by its hash.
func (r *bootstrapTokenRepository) GetByTokenHash(tokenHash string) (*user.BootstrapToken, error) {
	var token user.BootstrapToken
	err := r.db.Where("token_hash = ?", tokenHash).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// IsTokenUsed checks if a token hash has been used.
func (r *bootstrapTokenRepository) IsTokenUsed(tokenHash string) (bool, error) {
	var count int64
	err := r.db.Model(&user.BootstrapToken{}).Where("token_hash = ?", tokenHash).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

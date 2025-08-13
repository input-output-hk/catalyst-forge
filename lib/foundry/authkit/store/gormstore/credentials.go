package gormstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CredentialStore is a GORM-based implementation of store.CredentialStore.
type CredentialStore struct {
	db *gorm.DB
}

// NewCredentialStore creates a new GORM-based credential store.
func NewCredentialStore(db *gorm.DB) *CredentialStore {
	return &CredentialStore{db: db}
}

// Add stores a new credential for a user.
func (s *CredentialStore) Add(ctx context.Context, cred *domain.Credential) error {
	var transportsStr string
	if cred.Transports != nil && len(cred.Transports) > 0 {
		transportsJSON, err := json.Marshal(cred.Transports)
		if err != nil {
			return err
		}
		transportsStr = string(transportsJSON)
	}

	dbCred := &Credential{
		ID:         cred.ID,
		UserID:     cred.UserID,
		PublicKey:  cred.PublicKey,
		AAGUID:     cred.AAGUID,
		DeviceName: cred.DeviceName,
		RK:         cred.RK,
		Transports: transportsStr,
		SignCount:  cred.SignCount,
		CreatedAt:  cred.CreatedAt,
		LastUsedAt: cred.LastUsedAt,
		Revoked:    cred.Revoked,
	}

	if err := s.db.WithContext(ctx).Create(dbCred).Error; err != nil {
		return err
	}

	return nil
}

// GetByUser retrieves all credentials for a user.
func (s *CredentialStore) GetByUser(ctx context.Context, userID uuid.UUID) ([]domain.Credential, error) {
	var dbCreds []Credential
	
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND revoked = ?", userID, false).
		Find(&dbCreds).Error; err != nil {
		return nil, err
	}

	creds := make([]domain.Credential, 0, len(dbCreds))
	for _, dbCred := range dbCreds {
		domainCred, err := s.toDomain(&dbCred)
		if err != nil {
			return nil, err
		}
		creds = append(creds, *domainCred)
	}

	return creds, nil
}

// Get retrieves a specific credential by its ID.
func (s *CredentialStore) Get(ctx context.Context, id []byte) (*domain.Credential, error) {
	var dbCred Credential
	
	if err := s.db.WithContext(ctx).
		Where("id = ?", id).
		First(&dbCred).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("credential not found")
		}
		return nil, err
	}

	if dbCred.Revoked {
		return nil, errors.New("credential revoked")
	}

	return s.toDomain(&dbCred)
}

// UpdateOnAssertion updates the sign count and last used time after successful authentication.
func (s *CredentialStore) UpdateOnAssertion(ctx context.Context, id []byte, signCount uint32, lastUsed time.Time) error {
	result := s.db.WithContext(ctx).Model(&Credential{}).
		Where("id = ? AND revoked = ?", id, false).
		Updates(map[string]interface{}{
			"sign_count":   signCount,
			"last_used_at": lastUsed,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("credential not found or revoked")
	}

	return nil
}

// Revoke marks a credential as revoked.
func (s *CredentialStore) Revoke(ctx context.Context, id []byte) error {
	result := s.db.WithContext(ctx).Model(&Credential{}).
		Where("id = ?", id).
		Update("revoked", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("credential not found")
	}

	return nil
}

// toDomain converts a database model to a domain entity.
func (s *CredentialStore) toDomain(cred *Credential) (*domain.Credential, error) {
	var transports []string
	if cred.Transports != "" {
		if err := json.Unmarshal([]byte(cred.Transports), &transports); err != nil {
			return nil, err
		}
	} else {
		transports = []string{}
	}

	return &domain.Credential{
		ID:         cred.ID,
		UserID:     cred.UserID,
		PublicKey:  cred.PublicKey,
		AAGUID:     cred.AAGUID,
		DeviceName: cred.DeviceName,
		RK:         cred.RK,
		Transports: transports,
		SignCount:  cred.SignCount,
		CreatedAt:  cred.CreatedAt,
		LastUsedAt: cred.LastUsedAt,
		Revoked:    cred.Revoked,
	}, nil
}
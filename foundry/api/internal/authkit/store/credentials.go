package store

import (
	"context"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/google/uuid"
)

// CredentialStore handles WebAuthn credential persistence.
type CredentialStore interface {
	// Add stores a new credential for a user.
	Add(ctx context.Context, cred *domain.Credential) error
	
	// GetByUser retrieves all credentials for a user.
	GetByUser(ctx context.Context, userID uuid.UUID) ([]domain.Credential, error)
	
	// Get retrieves a specific credential by its ID.
	Get(ctx context.Context, id []byte) (*domain.Credential, error)
	
	// UpdateOnAssertion updates the sign count and last used time after successful authentication.
	UpdateOnAssertion(ctx context.Context, id []byte, signCount uint32, lastUsed time.Time) error
	
	// Revoke marks a credential as revoked.
	Revoke(ctx context.Context, id []byte) error
}
package inmemory

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/google/uuid"
)

// CredentialStore is an in-memory implementation of store.CredentialStore.
type CredentialStore struct {
	mu          sync.RWMutex
	credentials map[string]*domain.Credential // Key is base64 of credential ID
	byUser      map[uuid.UUID][]*domain.Credential
}

// NewCredentialStore creates a new in-memory credential store.
func NewCredentialStore() *CredentialStore {
	return &CredentialStore{
		credentials: make(map[string]*domain.Credential),
		byUser:      make(map[uuid.UUID][]*domain.Credential),
	}
}

// credentialKey generates a map key from credential ID bytes.
func credentialKey(id []byte) string {
	return string(id)
}

// Add stores a new credential for a user.
func (s *CredentialStore) Add(ctx context.Context, cred *domain.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := credentialKey(cred.ID)
	if _, exists := s.credentials[key]; exists {
		return errors.New("credential already exists")
	}

	// Store credential
	credCopy := *cred
	s.credentials[key] = &credCopy

	// Add to user's credential list
	s.byUser[cred.UserID] = append(s.byUser[cred.UserID], &credCopy)

	return nil
}

// GetByUser retrieves all credentials for a user.
func (s *CredentialStore) GetByUser(ctx context.Context, userID uuid.UUID) ([]domain.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	creds := s.byUser[userID]
	result := make([]domain.Credential, 0, len(creds))
	
	for _, cred := range creds {
		if !cred.Revoked {
			credCopy := *cred
			result = append(result, credCopy)
		}
	}

	return result, nil
}

// Get retrieves a specific credential by its ID.
func (s *CredentialStore) Get(ctx context.Context, id []byte) (*domain.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cred, exists := s.credentials[credentialKey(id)]
	if !exists {
		return nil, errors.New("credential not found")
	}

	if cred.Revoked {
		return nil, errors.New("credential revoked")
	}

	// Return a copy
	credCopy := *cred
	return &credCopy, nil
}

// UpdateOnAssertion updates the sign count and last used time after successful authentication.
func (s *CredentialStore) UpdateOnAssertion(ctx context.Context, id []byte, signCount uint32, lastUsed time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cred, exists := s.credentials[credentialKey(id)]
	if !exists {
		return errors.New("credential not found")
	}

	if cred.Revoked {
		return errors.New("credential revoked")
	}

	cred.SignCount = signCount
	cred.LastUsedAt = lastUsed

	return nil
}

// Revoke marks a credential as revoked.
func (s *CredentialStore) Revoke(ctx context.Context, id []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cred, exists := s.credentials[credentialKey(id)]
	if !exists {
		return errors.New("credential not found")
	}

	cred.Revoked = true

	// Also update in byUser map
	for i, userCred := range s.byUser[cred.UserID] {
		if bytes.Equal(userCred.ID, id) {
			s.byUser[cred.UserID][i].Revoked = true
			break
		}
	}

	return nil
}
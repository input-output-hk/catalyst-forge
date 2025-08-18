package inmemory

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
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
func credentialKey(id []byte) string { return string(id) }

// Add stores a new credential for a user.
func (s *CredentialStore) Add(ctx context.Context, cred *domain.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := credentialKey(cred.ID)
	if _, exists := s.credentials[key]; exists {
		return errors.New("credential already exists")
	}
	cp := *cred
	s.credentials[key] = &cp
	s.byUser[cred.UserID] = append(s.byUser[cred.UserID], &cp)
	return nil
}

// GetByUser retrieves all credentials for a user.
func (s *CredentialStore) GetByUser(ctx context.Context, userID uuid.UUID) ([]domain.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	creds := s.byUser[userID]
	out := make([]domain.Credential, 0, len(creds))
	for _, c := range creds {
		if !c.Revoked {
			out = append(out, *c)
		}
	}
	return out, nil
}

// Get retrieves a specific credential by its ID.
func (s *CredentialStore) Get(ctx context.Context, id []byte) (*domain.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.credentials[credentialKey(id)]
	if !ok {
		return nil, errors.New("credential not found")
	}
	if c.Revoked {
		return nil, errors.New("credential revoked")
	}
	cp := *c
	return &cp, nil
}

// UpdateOnAssertion updates the sign count and last used time after successful authentication.
func (s *CredentialStore) UpdateOnAssertion(ctx context.Context, id []byte, signCount uint32, lastUsed time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.credentials[credentialKey(id)]
	if !ok {
		return errors.New("credential not found")
	}
	if c.Revoked {
		return errors.New("credential revoked")
	}
	c.SignCount = signCount
	c.LastUsedAt = lastUsed
	return nil
}

// Revoke marks a credential as revoked.
func (s *CredentialStore) Revoke(ctx context.Context, id []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.credentials[credentialKey(id)]
	if !ok {
		return errors.New("credential not found")
	}
	c.Revoked = true
	for i, uc := range s.byUser[c.UserID] {
		if bytes.Equal(uc.ID, id) {
			s.byUser[c.UserID][i].Revoked = true
			break
		}
	}
	return nil
}

// UpdateDeviceName updates the device name for a credential.
func (s *CredentialStore) UpdateDeviceName(_ context.Context, id []byte, userID uuid.UUID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.credentials[credentialKey(id)]
	if !ok {
		return errors.New("credential not found")
	}
	if c.Revoked {
		return errors.New("credential revoked")
	}
	if c.UserID != userID {
		return errors.New("credential does not belong to user")
	}
	c.DeviceName = name
	for _, uc := range s.byUser[userID] {
		if bytes.Equal(uc.ID, id) {
			uc.DeviceName = name
			break
		}
	}
	return nil
}

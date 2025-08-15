package inmemory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/google/uuid"
)

// RefreshStore is an in-memory implementation of store.RefreshStore.
type RefreshStore struct {
	mu       sync.RWMutex
	tokens   map[uuid.UUID]*domain.RefreshToken
	byHash   map[string]uuid.UUID      // hash -> token ID
	byFamily map[uuid.UUID][]uuid.UUID // family ID -> token IDs
	byUser   map[uuid.UUID][]uuid.UUID // user ID -> token IDs
	byDevice map[uuid.UUID][]uuid.UUID // device ID -> token IDs
}

// NewRefreshStore creates a new in-memory refresh token store.
func NewRefreshStore() *RefreshStore {
	return &RefreshStore{
		tokens:   make(map[uuid.UUID]*domain.RefreshToken),
		byHash:   make(map[string]uuid.UUID),
		byFamily: make(map[uuid.UUID][]uuid.UUID),
		byUser:   make(map[uuid.UUID][]uuid.UUID),
		byDevice: make(map[uuid.UUID][]uuid.UUID),
	}
}

// hashKey converts a byte hash to a string key.
func hashKey(hash []byte) string {
	return string(hash)
}

// CreateFamily creates a new refresh token family for initial login.
func (s *RefreshStore) CreateFamily(ctx context.Context, userID uuid.UUID, sessionVersion int64, tokenHash []byte, expiresAt time.Time) (familyID uuid.UUID, tokenID uuid.UUID, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	familyID = uuid.New()
	tokenID = uuid.New()

	// Create token
	token := &domain.RefreshToken{
		ID:             tokenID,
		FamilyID:       familyID,
		UserID:         userID,
		Hash:           make([]byte, len(tokenHash)),
		SessionVersion: sessionVersion,
		CreatedAt:      time.Now(),
		ExpiresAt:      expiresAt,
	}
	copy(token.Hash, tokenHash)

	// Store token
	s.tokens[tokenID] = token
	s.byHash[hashKey(tokenHash)] = tokenID
	s.byFamily[familyID] = []uuid.UUID{tokenID}
	s.byUser[userID] = append(s.byUser[userID], tokenID)

	return familyID, tokenID, nil
}

// CreateFamilyWithDevice creates a new refresh token family for CLI device login.
func (s *RefreshStore) CreateFamilyWithDevice(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID, sessionVersion int64, tokenHash []byte, expiresAt time.Time) (familyID uuid.UUID, tokenID uuid.UUID, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	familyID = uuid.New()
	tokenID = uuid.New()

	// Create token with device ID
	token := &domain.RefreshToken{
		ID:             tokenID,
		FamilyID:       familyID,
		UserID:         userID,
		DeviceID:       &deviceID,
		Hash:           make([]byte, len(tokenHash)),
		SessionVersion: sessionVersion,
		CreatedAt:      time.Now(),
		ExpiresAt:      expiresAt,
	}
	copy(token.Hash, tokenHash)

	// Store token
	s.tokens[tokenID] = token
	s.byHash[hashKey(tokenHash)] = tokenID
	s.byFamily[familyID] = []uuid.UUID{tokenID}
	s.byUser[userID] = append(s.byUser[userID], tokenID)
	s.byDevice[deviceID] = append(s.byDevice[deviceID], tokenID)

	return familyID, tokenID, nil
}

// Rotate rotates a refresh token to a new one within the same family.
//
// This marks the previous token as rotated and creates a new one.
func (s *RefreshStore) Rotate(ctx context.Context, prevTokenID uuid.UUID, newHash []byte, now time.Time, expiresAt time.Time) (newTokenID uuid.UUID, familyID uuid.UUID, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get previous token
	prevToken, exists := s.tokens[prevTokenID]
	if !exists {
		return uuid.Nil, uuid.Nil, errors.New("token not found")
	}

	if prevToken.RotatedAt != nil {
		return uuid.Nil, uuid.Nil, errors.New("token already rotated")
	}

	if prevToken.RevokedAt != nil {
		return uuid.Nil, uuid.Nil, errors.New("token revoked")
	}

	// Mark previous token as rotated
	prevToken.RotatedAt = &now

	// Create new token
	newTokenID = uuid.New()
	newToken := &domain.RefreshToken{
		ID:             newTokenID,
		FamilyID:       prevToken.FamilyID,
		UserID:         prevToken.UserID,
		DeviceID:       prevToken.DeviceID, // Preserve device ID if present
		Hash:           make([]byte, len(newHash)),
		SessionVersion: prevToken.SessionVersion,
		CreatedAt:      now,
		ExpiresAt:      expiresAt,
	}
	copy(newToken.Hash, newHash)

	// Store new token
	s.tokens[newTokenID] = newToken
	s.byHash[hashKey(newHash)] = newTokenID
	s.byFamily[prevToken.FamilyID] = append(s.byFamily[prevToken.FamilyID], newTokenID)
	s.byUser[prevToken.UserID] = append(s.byUser[prevToken.UserID], newTokenID)
	
	// Add to device index if device ID is present
	if prevToken.DeviceID != nil {
		s.byDevice[*prevToken.DeviceID] = append(s.byDevice[*prevToken.DeviceID], newTokenID)
	}

	// Remove old hash mapping
	delete(s.byHash, hashKey(prevToken.Hash))

	return newTokenID, prevToken.FamilyID, nil
}

// GetByID retrieves a refresh token by its ID.
func (s *RefreshStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, exists := s.tokens[id]
	if !exists {
		return nil, errors.New("token not found")
	}

	// Return a copy
	tokenCopy := *token
	return &tokenCopy, nil
}

// GetByHash retrieves a refresh token by its hash.
func (s *RefreshStore) GetByHash(ctx context.Context, hash []byte) (*domain.RefreshToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokenID, exists := s.byHash[hashKey(hash)]
	if !exists {
		return nil, errors.New("token not found")
	}

	token := s.tokens[tokenID]
	// Return a copy
	tokenCopy := *token
	return &tokenCopy, nil
}

// RevokeToken revokes a specific refresh token.
func (s *RefreshStore) RevokeToken(ctx context.Context, id uuid.UUID, reason string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.tokens[id]
	if !exists {
		return errors.New("token not found")
	}

	token.RevokedAt = &at
	token.Reason = &reason

	// Remove from hash mapping
	delete(s.byHash, hashKey(token.Hash))

	return nil
}

// RevokeFamily revokes all tokens in a family.
//
// This is used for replay detection.
func (s *RefreshStore) RevokeFamily(ctx context.Context, familyID uuid.UUID, reason string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenIDs, exists := s.byFamily[familyID]
	if !exists {
		return nil // No tokens in family
	}

	for _, tokenID := range tokenIDs {
		if token, exists := s.tokens[tokenID]; exists {
			token.RevokedAt = &at
			token.Reason = &reason
			// Remove from hash mapping
			delete(s.byHash, hashKey(token.Hash))
		}
	}

	return nil
}

// RevokeUserTokens revokes all refresh tokens for a user.
func (s *RefreshStore) RevokeUserTokens(ctx context.Context, userID uuid.UUID, reason string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenIDs, exists := s.byUser[userID]
	if !exists {
		return nil // No tokens for user
	}

	for _, tokenID := range tokenIDs {
		if token, exists := s.tokens[tokenID]; exists && token.RevokedAt == nil {
			token.RevokedAt = &at
			token.Reason = &reason
			// Remove from hash mapping
			delete(s.byHash, hashKey(token.Hash))
		}
	}

	return nil
}

// RevokeDeviceTokens revokes all refresh tokens for a device.
func (s *RefreshStore) RevokeDeviceTokens(ctx context.Context, deviceID uuid.UUID, reason string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenIDs, exists := s.byDevice[deviceID]
	if !exists {
		return nil // No tokens for device
	}

	for _, tokenID := range tokenIDs {
		if token, exists := s.tokens[tokenID]; exists && token.RevokedAt == nil {
			token.RevokedAt = &at
			token.Reason = &reason
			// Remove from hash mapping
			delete(s.byHash, hashKey(token.Hash))
		}
	}

	return nil
}

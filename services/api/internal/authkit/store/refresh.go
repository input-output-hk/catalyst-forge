package store

import (
	"context"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/google/uuid"
)

// RefreshStore handles refresh token persistence with rotation tracking.
type RefreshStore interface {
	// CreateFamily creates a new refresh token family for initial login.
	CreateFamily(ctx context.Context, userID uuid.UUID, sessionVersion int64, tokenHash []byte, expiresAt time.Time) (familyID uuid.UUID, tokenID uuid.UUID, err error)
	
	// CreateFamilyWithDevice creates a new refresh token family for CLI device login.
	CreateFamilyWithDevice(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID, sessionVersion int64, tokenHash []byte, expiresAt time.Time) (familyID uuid.UUID, tokenID uuid.UUID, err error)
	
	// Rotate rotates a refresh token to a new one within the same family.
	//
	// This marks the previous token as rotated and creates a new one.
	Rotate(ctx context.Context, prevTokenID uuid.UUID, newHash []byte, now time.Time, expiresAt time.Time) (newTokenID uuid.UUID, familyID uuid.UUID, err error)
	
	// GetByID retrieves a refresh token by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error)
	
	// GetByHash retrieves a refresh token by its hash.
	GetByHash(ctx context.Context, hash []byte) (*domain.RefreshToken, error)
	
	// RevokeToken revokes a specific refresh token.
	RevokeToken(ctx context.Context, id uuid.UUID, reason string, at time.Time) error
	
	// RevokeFamily revokes all tokens in a family.
	//
	// This is used for replay detection.
	RevokeFamily(ctx context.Context, familyID uuid.UUID, reason string, at time.Time) error
	
	// RevokeUserTokens revokes all refresh tokens for a user.
	RevokeUserTokens(ctx context.Context, userID uuid.UUID, reason string, at time.Time) error
	
	// RevokeDeviceTokens revokes all refresh tokens for a device.
	RevokeDeviceTokens(ctx context.Context, deviceID uuid.UUID, reason string, at time.Time) error
}
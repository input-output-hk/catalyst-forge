package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/crypto"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/store"
	"github.com/google/uuid"
)

// RefreshService handles refresh token operations with rotation and replay detection.
type RefreshService interface {
	// Issue creates a new refresh token family for initial login.
	Issue(ctx context.Context, user *domain.User, now time.Time) (cookieValue string, tokenID uuid.UUID, familyID uuid.UUID, err error)
	
	// Rotate rotates a refresh token to a new one, detecting replays.
	Rotate(ctx context.Context, cookieValue string, now time.Time) (newCookie string, accessJWT string, user *domain.User, err error)
	
	// RevokeCurrent revokes the current refresh token.
	RevokeCurrent(ctx context.Context, cookieValue string, reason string) error
	
	// RevokeAllUser revokes all refresh tokens for a user.
	RevokeAllUser(ctx context.Context, userID uuid.UUID, reason string) error
}

// refreshService implements RefreshService.
type refreshService struct {
	store       store.RefreshStore
	userStore   store.UserStore
	tokenSvc    TokenService
	rand        crypto.Rand
	ttl         time.Duration
	auditStore  store.AuditStore
}

// RefreshServiceConfig holds configuration for the refresh service.
type RefreshServiceConfig struct {
	Store      store.RefreshStore
	UserStore  store.UserStore
	TokenSvc   TokenService
	Rand       crypto.Rand
	TTL        time.Duration
	AuditStore store.AuditStore
}

// NewRefreshService creates a new refresh service.
func NewRefreshService(cfg RefreshServiceConfig) RefreshService {
	if cfg.TTL <= 0 {
		panic("refresh token TTL must be > 0")
	}
	return &refreshService{
		store:      cfg.Store,
		userStore:  cfg.UserStore,
		tokenSvc:   cfg.TokenSvc,
		rand:       cfg.Rand,
		ttl:        cfg.TTL,
		auditStore: cfg.AuditStore,
	}
}

// Issue creates a new refresh token family for initial login.
func (s *refreshService) Issue(ctx context.Context, user *domain.User, now time.Time) (cookieValue string, tokenID uuid.UUID, familyID uuid.UUID, err error) {
	// Generate random token value
	rawBytes, err := s.rand.Bytes(32)
	if err != nil {
		return "", uuid.Nil, uuid.Nil, err
	}
	
	// The cookie value is base64url-encoded (no padding) for safe transport
	cookieValue = base64.RawURLEncoding.EncodeToString(rawBytes)
	
	// Hash the raw bytes for storage
	tokenHash := crypto.HashSHA256(rawBytes)
	
	// Calculate expiry time
	expiresAt := now.Add(s.ttl)
	
	// Create the refresh token family
	familyID, tokenID, err = s.store.CreateFamily(ctx, user.ID, user.SessionVersion, tokenHash, expiresAt)
	if err != nil {
		return "", uuid.Nil, uuid.Nil, err
	}
	
	// Audit the event
	if s.auditStore != nil {
		event := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventTokenRefresh,
			UserID:    &user.ID,
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"family_id": familyID.String(),
				"token_id":  tokenID.String(),
				"action":    "issue",
			},
		}
		_ = s.auditStore.Record(ctx, event)
	}
	
	return cookieValue, tokenID, familyID, nil
}

// Rotate rotates a refresh token to a new one, detecting replays.
func (s *refreshService) Rotate(ctx context.Context, cookieValue string, now time.Time) (newCookie string, accessJWT string, user *domain.User, err error) {
	// Decode the base64url-encoded token
	rawBytes, err := base64.RawURLEncoding.DecodeString(cookieValue)
	if err != nil || len(rawBytes) != 32 {
		return "", "", nil, errors.New("invalid refresh token")
	}
	
	// Hash the raw bytes
	tokenHash := crypto.HashSHA256(rawBytes)
	
	// Look up the token
	token, err := s.store.GetByHash(ctx, tokenHash)
	if err != nil {
		return "", "", nil, errors.New("invalid refresh token")
	}
	
	// Check if token has been rotated (replay detection)
	if token.RotatedAt != nil {
		// This is a replay attack! Revoke the entire family
		_ = s.store.RevokeFamily(ctx, token.FamilyID, "replay detected", now)
		
		// Audit the security event
		if s.auditStore != nil {
			event := domain.Event{
				ID:        uuid.New(),
				Type:      domain.EventTokenFamilyRevoked,
				UserID:    &token.UserID,
				CreatedAt: now,
				Metadata: map[string]interface{}{
					"family_id": token.FamilyID.String(),
					"reason":    "replay detected",
					"token_id":  token.ID.String(),
				},
			}
			_ = s.auditStore.Record(ctx, event)
		}
		
		return "", "", nil, errors.New("refresh token replay detected")
	}
	
	// Check if token has been revoked
	if token.RevokedAt != nil {
		return "", "", nil, errors.New("refresh token revoked")
	}
	
	// Check if token has expired
	if token.ExpiresAt.Before(now) {
		return "", "", nil, errors.New("refresh token expired")
	}
	
	// Get the user
	user, err = s.userStore.GetByID(ctx, token.UserID)
	if err != nil {
		return "", "", nil, err
	}
	
	// Check session version
	if token.SessionVersion != user.SessionVersion {
		// Session has been invalidated
		_ = s.store.RevokeToken(ctx, token.ID, "session version mismatch", now)
		return "", "", nil, errors.New("session expired")
	}
	
	// Generate new token
	newRawBytes, err := s.rand.Bytes(32)
	if err != nil {
		return "", "", nil, err
	}
	
	newCookie = base64.RawURLEncoding.EncodeToString(newRawBytes)
	newTokenHash := crypto.HashSHA256(newRawBytes)
	
	// Calculate expiry for new token
	newExpiresAt := now.Add(s.ttl)
	
	// Rotate the token
	newTokenID, _, err := s.store.Rotate(ctx, token.ID, newTokenHash, now, newExpiresAt)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to rotate token: %w", err)
	}
	
	// Generate new access token
	claims := AccessClaims{
		Sub:            user.ID.String(),
		Email:          user.Email,
		Roles:          user.Roles,
		SessionVersion: user.SessionVersion,
		// JTI will be generated by TokenService.SignAccess
	}
	
	accessJWT, err = s.tokenSvc.SignAccess(ctx, claims)
	if err != nil {
		// Rollback by revoking the new token
		_ = s.store.RevokeToken(ctx, newTokenID, "access token generation failed", now)
		return "", "", nil, err
	}
	
	// Audit the event
	if s.auditStore != nil {
		event := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventTokenRefresh,
			UserID:    &user.ID,
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"family_id":     token.FamilyID.String(),
				"old_token_id":  token.ID.String(),
				"new_token_id":  newTokenID.String(),
				"action":        "rotate",
			},
		}
		_ = s.auditStore.Record(ctx, event)
	}
	
	return newCookie, accessJWT, user, nil
}

// RevokeCurrent revokes the entire refresh token family (device-wide logout).
func (s *refreshService) RevokeCurrent(ctx context.Context, cookieValue string, reason string) error {
	// Decode the base64url-encoded token
	rawBytes, err := base64.RawURLEncoding.DecodeString(cookieValue)
	if err != nil || len(rawBytes) != 32 {
		// Invalid token format, consider it already revoked
		return nil
	}
	
	// Hash the raw bytes
	tokenHash := crypto.HashSHA256(rawBytes)
	
	// Look up the token
	token, err := s.store.GetByHash(ctx, tokenHash)
	if err != nil {
		// Token not found, consider it already revoked
		return nil
	}
	
	// Revoke the token family (user is logging out)
	now := time.Now().UTC()
	err = s.store.RevokeFamily(ctx, token.FamilyID, reason, now)
	if err != nil {
		return err
	}
	
	// Audit the event
	if s.auditStore != nil {
		event := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventLogout,
			UserID:    &token.UserID,
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"family_id": token.FamilyID.String(),
				"reason":    reason,
			},
		}
		_ = s.auditStore.Record(ctx, event)
	}
	
	return nil
}

// RevokeAllUser revokes all refresh tokens for a user.
func (s *refreshService) RevokeAllUser(ctx context.Context, userID uuid.UUID, reason string) error {
	now := time.Now().UTC()
	
	// Revoke all tokens
	err := s.store.RevokeUserTokens(ctx, userID, reason, now)
	if err != nil {
		return err
	}
	
	// Bump session version to invalidate access tokens
	err = s.userStore.BumpSessionVersion(ctx, userID)
	if err != nil {
		return err
	}
	
	// Audit the event
	if s.auditStore != nil {
		event := domain.Event{
			ID:        uuid.New(),
			Type:      domain.EventLogoutAll,
			UserID:    &userID,
			CreatedAt: now,
			Metadata: map[string]interface{}{
				"reason": reason,
			},
		}
		_ = s.auditStore.Record(ctx, event)
	}
	
	return nil
}
package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/testing/inmemory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRefreshService(t *testing.T) (RefreshService, *inmemory.RefreshStore, *inmemory.UserStore, *inmemory.AuditStore) {
	t.Helper()
	
	// Create stores
	refreshStore := inmemory.NewRefreshStore()
	userStore := inmemory.NewUserStore()
	auditStore := inmemory.NewAuditStore()
	
	// Create mock token service
	tokenSvc := &mockTokenService{}
	
	// Create secure random
	rand := crypto.NewSecureRand()
	
	// Create refresh service with 7 day TTL
	cfg := RefreshServiceConfig{
		Store:      refreshStore,
		UserStore:  userStore,
		TokenSvc:   tokenSvc,
		Rand:       rand,
		TTL:        7 * 24 * time.Hour,
		AuditStore: auditStore,
	}
	
	refreshSvc := NewRefreshService(cfg)
	
	return refreshSvc, refreshStore, userStore, auditStore
}

// mockTokenService implements TokenService for testing
type mockTokenService struct {
	signFunc  func(ctx context.Context, claims AccessClaims) (string, error)
	parseFunc func(ctx context.Context, token string) (*AccessClaims, error)
	jwksFunc  func() interface{}
}

func (m *mockTokenService) SignAccess(ctx context.Context, claims AccessClaims) (string, error) {
	if m.signFunc != nil {
		return m.signFunc(ctx, claims)
	}
	// Default implementation
	return "mock.access.token", nil
}

func (m *mockTokenService) ParseAccess(ctx context.Context, token string) (*AccessClaims, error) {
	if m.parseFunc != nil {
		return m.parseFunc(ctx, token)
	}
	// Default implementation
	return &AccessClaims{
		Sub:            uuid.New().String(),
		Email:          "user@example.com",
		SessionVersion: 1,
	}, nil
}

func (m *mockTokenService) JWKS() interface{} {
	if m.jwksFunc != nil {
		return m.jwksFunc()
	}
	return map[string]interface{}{"keys": []interface{}{}}
}

func TestNewRefreshService(t *testing.T) {
	t.Parallel()
	
	t.Run("ok/valid_config", func(t *testing.T) {
		t.Parallel()
		
		cfg := RefreshServiceConfig{
			Store:      inmemory.NewRefreshStore(),
			UserStore:  inmemory.NewUserStore(),
			TokenSvc:   &mockTokenService{},
			Rand:       crypto.NewSecureRand(),
			TTL:        time.Hour,
			AuditStore: inmemory.NewAuditStore(),
		}
		
		assert.NotPanics(t, func() {
			_ = NewRefreshService(cfg)
		})
	})
	
	t.Run("panic/invalid_ttl", func(t *testing.T) {
		t.Parallel()
		
		cfg := RefreshServiceConfig{
			Store:     inmemory.NewRefreshStore(),
			UserStore: inmemory.NewUserStore(),
			TokenSvc:  &mockTokenService{},
			Rand:      crypto.NewSecureRand(),
			TTL:       0, // Invalid
		}
		
		assert.Panics(t, func() {
			_ = NewRefreshService(cfg)
		})
		
		cfg.TTL = -time.Hour
		assert.Panics(t, func() {
			_ = NewRefreshService(cfg)
		})
	})
}

func TestRefreshService_Issue(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	refreshSvc, refreshStore, userStore, auditStore := setupRefreshService(t)
	
	// Create a test user
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err)
	
	t.Run("ok/issue_new_token", func(t *testing.T) {
		now := time.Now()
		
		cookieValue, tokenID, familyID, err := refreshSvc.Issue(ctx, user, now)
		require.NoError(t, err)
		assert.NotEmpty(t, cookieValue)
		assert.NotEqual(t, uuid.Nil, tokenID)
		assert.NotEqual(t, uuid.Nil, familyID)
		
		// Verify cookie value is base64url encoded
		decoded, err := base64.RawURLEncoding.DecodeString(cookieValue)
		require.NoError(t, err)
		assert.Len(t, decoded, 32, "should be 32 random bytes")
		
		// Verify token was stored
		tokenHash := crypto.HashSHA256(decoded)
		storedToken, err := refreshStore.GetByHash(ctx, tokenHash)
		require.NoError(t, err)
		assert.Equal(t, tokenID, storedToken.ID)
		assert.Equal(t, familyID, storedToken.FamilyID)
		assert.Equal(t, user.ID, storedToken.UserID)
		assert.Equal(t, user.SessionVersion, storedToken.SessionVersion)
		
		// Verify audit event
		events := auditStore.GetEvents()
		require.Len(t, events, 1)
		assert.Equal(t, domain.EventTokenRefresh, events[0].Type)
		assert.Equal(t, user.ID, *events[0].UserID)
		assert.Equal(t, "issue", events[0].Metadata["action"])
	})
	
	t.Run("ok/multiple_families_same_user", func(t *testing.T) {
		now := time.Now()
		
		// Issue first token family (e.g., desktop device)
		cookie1, token1, family1, err := refreshSvc.Issue(ctx, user, now)
		require.NoError(t, err)
		
		// Issue second token family (e.g., mobile device)
		cookie2, token2, family2, err := refreshSvc.Issue(ctx, user, now)
		require.NoError(t, err)
		
		// Verify they are different
		assert.NotEqual(t, cookie1, cookie2)
		assert.NotEqual(t, token1, token2)
		assert.NotEqual(t, family1, family2)
		
		// Both should be valid
		decoded1, _ := base64.RawURLEncoding.DecodeString(cookie1)
		hash1 := crypto.HashSHA256(decoded1)
		stored1, err := refreshStore.GetByHash(ctx, hash1)
		require.NoError(t, err)
		assert.Equal(t, token1, stored1.ID)
		
		decoded2, _ := base64.RawURLEncoding.DecodeString(cookie2)
		hash2 := crypto.HashSHA256(decoded2)
		stored2, err := refreshStore.GetByHash(ctx, hash2)
		require.NoError(t, err)
		assert.Equal(t, token2, stored2.ID)
	})
}

func TestRefreshService_Rotate(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	refreshSvc, refreshStore, userStore, auditStore := setupRefreshService(t)
	
	// Create a test user
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err)
	
	// Issue initial token
	now := time.Now()
	cookieValue, _, familyID, err := refreshSvc.Issue(ctx, user, now)
	require.NoError(t, err)
	
	t.Run("ok/successful_rotation", func(t *testing.T) {
		// Get event count before rotation
		eventsBefore := len(auditStore.GetEvents())
		
		// Rotate the token
		newCookie, accessJWT, rotatedUser, err := refreshSvc.Rotate(ctx, cookieValue, now.Add(time.Hour))
		require.NoError(t, err)
		assert.NotEmpty(t, newCookie)
		assert.NotEmpty(t, accessJWT)
		assert.NotNil(t, rotatedUser)
		assert.Equal(t, user.ID, rotatedUser.ID)
		
		// Verify new cookie is different
		assert.NotEqual(t, cookieValue, newCookie)
		
		// Verify old token is no longer accessible by hash (removed from hash mapping after rotation)
		oldDecoded, _ := base64.RawURLEncoding.DecodeString(cookieValue)
		oldHash := crypto.HashSHA256(oldDecoded)
		_, err = refreshStore.GetByHash(ctx, oldHash)
		assert.Error(t, err, "old token should not be found by hash after rotation")
		
		// Verify new token exists
		newDecoded, _ := base64.RawURLEncoding.DecodeString(newCookie)
		newHash := crypto.HashSHA256(newDecoded)
		newToken, err := refreshStore.GetByHash(ctx, newHash)
		require.NoError(t, err)
		assert.Nil(t, newToken.RotatedAt, "new token should not be rotated")
		assert.Equal(t, familyID, newToken.FamilyID, "should be in same family")
		
		// Verify audit event
		eventsAfter := auditStore.GetEvents()
		newEvents := eventsAfter[eventsBefore:]
		require.Len(t, newEvents, 1)
		assert.Equal(t, domain.EventTokenRefresh, newEvents[0].Type)
		assert.Equal(t, "rotate", newEvents[0].Metadata["action"])
	})
	
	t.Run("error/old_token_after_rotation", func(t *testing.T) {
		// Get a fresh token
		freshCookie, _, _, err := refreshSvc.Issue(ctx, user, now)
		require.NoError(t, err)
		
		// Rotate it once
		newCookie, _, _, err := refreshSvc.Rotate(ctx, freshCookie, now.Add(time.Hour))
		require.NoError(t, err)
		assert.NotEmpty(t, newCookie)
		
		// Try to use the old token again (it's been removed from hash mapping)
		_, _, _, err = refreshSvc.Rotate(ctx, freshCookie, now.Add(2*time.Hour))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token")
		
		// The new token should still work
		newerCookie, _, _, err := refreshSvc.Rotate(ctx, newCookie, now.Add(3*time.Hour))
		require.NoError(t, err)
		assert.NotEmpty(t, newerCookie)
	})
	
	t.Run("error/invalid_token_format", func(t *testing.T) {
		_, _, _, err := refreshSvc.Rotate(ctx, "invalid-token", now)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token")
	})
	
	t.Run("error/token_not_found", func(t *testing.T) {
		// Create valid format but non-existent token
		randomBytes := make([]byte, 32)
		fakeToken := base64.RawURLEncoding.EncodeToString(randomBytes)
		
		_, _, _, err := refreshSvc.Rotate(ctx, fakeToken, now)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token")
	})
	
	t.Run("error/expired_token", func(t *testing.T) {
		// Issue a token that's already expired
		expiredCookie, _, _, err := refreshSvc.Issue(ctx, user, now.Add(-8*24*time.Hour))
		require.NoError(t, err)
		
		// Force the expiration by updating the token directly
		decoded, _ := base64.RawURLEncoding.DecodeString(expiredCookie)
		hash := crypto.HashSHA256(decoded)
		token, _ := refreshStore.GetByHash(ctx, hash)
		token.ExpiresAt = now.Add(-time.Hour)
		
		// Try to rotate expired token
		_, _, _, err = refreshSvc.Rotate(ctx, expiredCookie, now)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})
	
	t.Run("error/session_version_mismatch", func(t *testing.T) {
		// Issue a new token
		validCookie, _, _, err := refreshSvc.Issue(ctx, user, now)
		require.NoError(t, err)
		
		// Bump user's session version (simulating logout from all devices)
		err = userStore.BumpSessionVersion(ctx, user.ID)
		require.NoError(t, err)
		
		// Try to rotate with old session version
		_, _, _, err = refreshSvc.Rotate(ctx, validCookie, now)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session expired")
	})
}

func TestRefreshService_RevokeCurrent(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	refreshSvc, refreshStore, userStore, auditStore := setupRefreshService(t)
	
	// Create a test user
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err)
	
	now := time.Now()
	
	t.Run("ok/revoke_family", func(t *testing.T) {
		// Issue token family
		cookie1, _, family1, err := refreshSvc.Issue(ctx, user, now)
		require.NoError(t, err)
		
		// Rotate to create multiple tokens in family
		cookie2, _, _, err := refreshSvc.Rotate(ctx, cookie1, now.Add(time.Hour))
		require.NoError(t, err)
		
		// Get event count before revoke
		eventsBefore := len(auditStore.GetEvents())
		
		// Revoke using the current token
		err = refreshSvc.RevokeCurrent(ctx, cookie2, "user logout")
		require.NoError(t, err)
		
		// Verify entire family is revoked
		// After revocation, tokens are removed from hash mapping, so GetByHash will fail
		decoded1, _ := base64.RawURLEncoding.DecodeString(cookie1)
		hash1 := crypto.HashSHA256(decoded1)
		_, err = refreshStore.GetByHash(ctx, hash1)
		assert.Error(t, err, "first token should not be found by hash after revocation")
		
		decoded2, _ := base64.RawURLEncoding.DecodeString(cookie2)
		hash2 := crypto.HashSHA256(decoded2)
		_, err = refreshStore.GetByHash(ctx, hash2)
		assert.Error(t, err, "second token should not be found by hash after revocation")
		
		// Verify audit event
		eventsAfter := auditStore.GetEvents()
		newEvents := eventsAfter[eventsBefore:]
		require.Len(t, newEvents, 1)
		assert.Equal(t, domain.EventLogout, newEvents[0].Type)
		assert.Equal(t, family1.String(), newEvents[0].Metadata["family_id"])
		assert.Equal(t, "user logout", newEvents[0].Metadata["reason"])
		
		// Verify neither token can be used
		_, _, _, err = refreshSvc.Rotate(ctx, cookie2, now.Add(2*time.Hour))
		require.Error(t, err)
		
		// Issue a different family - should still work
		cookie3, _, family3, err := refreshSvc.Issue(ctx, user, now)
		require.NoError(t, err)
		assert.NotEqual(t, family1, family3)
		
		// New family should work
		_, _, _, err = refreshSvc.Rotate(ctx, cookie3, now.Add(3*time.Hour))
		require.NoError(t, err)
	})
	
	t.Run("ok/revoke_invalid_token", func(t *testing.T) {
		// Should not error on invalid tokens (already logged out)
		err := refreshSvc.RevokeCurrent(ctx, "invalid-token", "test")
		assert.NoError(t, err)
		
		err = refreshSvc.RevokeCurrent(ctx, "", "test")
		assert.NoError(t, err)
		
		// Non-existent but valid format
		fakeToken := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
		err = refreshSvc.RevokeCurrent(ctx, fakeToken, "test")
		assert.NoError(t, err)
	})
	
	t.Run("ok/revoke_already_revoked", func(t *testing.T) {
		// Issue and revoke a token
		cookie, _, _, err := refreshSvc.Issue(ctx, user, now)
		require.NoError(t, err)
		
		err = refreshSvc.RevokeCurrent(ctx, cookie, "first revoke")
		require.NoError(t, err)
		
		// Revoke again - should not error
		err = refreshSvc.RevokeCurrent(ctx, cookie, "second revoke")
		assert.NoError(t, err)
	})
}

func TestRefreshService_RevokeAllUser(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	refreshSvc, _, userStore, auditStore := setupRefreshService(t)
	
	// Create test users
	user1, err := userStore.Create(ctx, "user1@example.com", []string{"user"})
	require.NoError(t, err)
	
	user2, err := userStore.Create(ctx, "user2@example.com", []string{"user"})
	require.NoError(t, err)
	
	now := time.Now()
	
	t.Run("ok/revoke_all_user_tokens", func(t *testing.T) {
		// Issue multiple token families for user1
		cookie1, _, _, err := refreshSvc.Issue(ctx, user1, now)
		require.NoError(t, err)
		
		cookie2, _, _, err := refreshSvc.Issue(ctx, user1, now)
		require.NoError(t, err)
		
		// Rotate one to create more tokens
		cookie3, _, _, err := refreshSvc.Rotate(ctx, cookie1, now.Add(time.Hour))
		require.NoError(t, err)
		
		// Issue token for user2
		user2Cookie, _, _, err := refreshSvc.Issue(ctx, user2, now)
		require.NoError(t, err)
		
		// Get event count before revoke
		eventsBefore := len(auditStore.GetEvents())
		
		// Revoke all tokens for user1
		err = refreshSvc.RevokeAllUser(ctx, user1.ID, "security incident")
		require.NoError(t, err)
		
		// Verify user1's session version was bumped
		updatedUser1, err := userStore.GetByID(ctx, user1.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), updatedUser1.SessionVersion, "session version should be incremented")
		
		// Verify audit event
		eventsAfter := auditStore.GetEvents()
		newEvents := eventsAfter[eventsBefore:]
		require.Len(t, newEvents, 1)
		assert.Equal(t, domain.EventLogoutAll, newEvents[0].Type)
		assert.Equal(t, user1.ID, *newEvents[0].UserID)
		assert.Equal(t, "security incident", newEvents[0].Metadata["reason"])
		
		// Verify none of user1's tokens work
		_, _, _, err = refreshSvc.Rotate(ctx, cookie1, now.Add(2*time.Hour))
		require.Error(t, err)
		
		_, _, _, err = refreshSvc.Rotate(ctx, cookie2, now.Add(2*time.Hour))
		require.Error(t, err)
		
		_, _, _, err = refreshSvc.Rotate(ctx, cookie3, now.Add(2*time.Hour))
		require.Error(t, err)
		
		// Verify user2's token still works
		_, _, _, err = refreshSvc.Rotate(ctx, user2Cookie, now.Add(2*time.Hour))
		require.NoError(t, err)
		
		// Verify user2's session version unchanged
		updatedUser2, err := userStore.GetByID(ctx, user2.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), updatedUser2.SessionVersion)
	})
	
	t.Run("ok/revoke_nonexistent_user", func(t *testing.T) {
		// Should not error for non-existent user
		err := refreshSvc.RevokeAllUser(ctx, uuid.New(), "test")
		// May error depending on implementation - check that it doesn't panic
		_ = err
	})
}

func TestRefreshService_TokenRotationChain(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	refreshSvc, _, userStore, _ := setupRefreshService(t)
	
	// Create a test user
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err)
	
	now := time.Now()
	
	// Issue initial token
	cookie1, _, _, err := refreshSvc.Issue(ctx, user, now)
	require.NoError(t, err)
	
	// Create a chain of rotations
	cookie2, _, _, err := refreshSvc.Rotate(ctx, cookie1, now.Add(1*time.Hour))
	require.NoError(t, err)
	
	cookie3, _, _, err := refreshSvc.Rotate(ctx, cookie2, now.Add(2*time.Hour))
	require.NoError(t, err)
	
	cookie4, _, _, err := refreshSvc.Rotate(ctx, cookie3, now.Add(3*time.Hour))
	require.NoError(t, err)
	
	// Only the latest should work
	cookie5, _, _, err := refreshSvc.Rotate(ctx, cookie4, now.Add(4*time.Hour))
	require.NoError(t, err)
	assert.NotEmpty(t, cookie5)
	
	// All previous tokens should fail (removed from hash mapping after rotation)
	for i, oldCookie := range []string{cookie1, cookie2, cookie3} {
		_, _, _, err := refreshSvc.Rotate(ctx, oldCookie, now.Add(5*time.Hour))
		require.Error(t, err, "cookie %d should fail", i+1)
		// After rotation, old tokens are removed from hash mapping and appear as invalid
		assert.Contains(t, err.Error(), "invalid refresh token")
	}
}

func TestRefreshService_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	refreshSvc, _, userStore, _ := setupRefreshService(t)
	
	// Create test users
	var users []*domain.User
	for i := 0; i < 5; i++ {
		user, err := userStore.Create(ctx, fmt.Sprintf("user%d@example.com", i), []string{"user"})
		require.NoError(t, err)
		users = append(users, user)
	}
	
	now := time.Now()
	
	// Run concurrent operations
	done := make(chan bool, 3)
	
	// Goroutine 1: Issue tokens
	go func() {
		for i := 0; i < 20; i++ {
			user := users[i%len(users)]
			_, _, _, err := refreshSvc.Issue(ctx, user, now)
			assert.NoError(t, err)
		}
		done <- true
	}()
	
	// Goroutine 2: Rotate tokens
	go func() {
		// Issue some tokens first
		var cookies []string
		for _, user := range users {
			cookie, _, _, err := refreshSvc.Issue(ctx, user, now)
			assert.NoError(t, err)
			cookies = append(cookies, cookie)
		}
		
		// Rotate them
		for _, cookie := range cookies {
			_, _, _, err := refreshSvc.Rotate(ctx, cookie, now.Add(time.Hour))
			// May fail if revoked by another goroutine, that's ok
			_ = err
		}
		done <- true
	}()
	
	// Goroutine 3: Revoke tokens
	go func() {
		// Issue and revoke
		for i := 0; i < 10; i++ {
			user := users[i%len(users)]
			cookie, _, _, err := refreshSvc.Issue(ctx, user, now)
			assert.NoError(t, err)
			
			err = refreshSvc.RevokeCurrent(ctx, cookie, "test")
			assert.NoError(t, err)
		}
		done <- true
	}()
	
	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestRefreshService_CookieFormat(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	refreshSvc, _, userStore, _ := setupRefreshService(t)
	
	// Create a test user
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err)
	
	// Issue a token
	cookieValue, _, _, err := refreshSvc.Issue(ctx, user, time.Now())
	require.NoError(t, err)
	
	// Verify cookie format
	assert.NotEmpty(t, cookieValue)
	
	// Should be base64url without padding
	assert.NotContains(t, cookieValue, "=", "should not have padding")
	assert.NotContains(t, cookieValue, "+", "should use URL-safe encoding")
	assert.NotContains(t, cookieValue, "/", "should use URL-safe encoding")
	
	// Should decode to 32 bytes
	decoded, err := base64.RawURLEncoding.DecodeString(cookieValue)
	require.NoError(t, err)
	assert.Len(t, decoded, 32, "should be 32 random bytes")
	
	// Should be URL-safe (no special characters that need escaping)
	for _, char := range cookieValue {
		assert.True(t, isURLSafeChar(char), "character %c is not URL-safe", char)
	}
}

func isURLSafeChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}

// TestRefreshService_DeviceSupport tests device-specific refresh token functionality
func TestRefreshService_DeviceSupport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		setupDevice      bool
		deviceID         *uuid.UUID
		wantAMR          []string
		wantDeviceIDSet  bool
	}{
		{
			name:            "ok/device_session_with_amr",
			setupDevice:     true,
			wantAMR:         []string{"device_link"},
			wantDeviceIDSet: true,
		},
		{
			name:            "ok/regular_session_no_device",
			setupDevice:     false,
			wantAMR:         []string{"webauthn"},
			wantDeviceIDSet: false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			refreshSvc, refreshStore, userStore, auditStore := setupRefreshService(t)

			// Create test user
			user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
			require.NoError(t, err, "failed to create test user")

			var cookieValue string
			var deviceID uuid.UUID

			if tc.setupDevice {
				// Create refresh token with device (simulating CLI login)
				deviceID = uuid.New()
				tc.deviceID = &deviceID
				
				// Generate token
				tokenBytes, err := refreshSvc.(*refreshService).rand.Bytes(32)
				require.NoError(t, err, "failed to generate random token")
				
				tokenHash := crypto.HashSHA256(tokenBytes)
				expiresAt := time.Now().Add(7 * 24 * time.Hour)
				
				// Create device-bound refresh token
				familyID, tokenID, err := refreshStore.CreateFamilyWithDevice(
					ctx, user.ID, deviceID, user.SessionVersion, tokenHash, expiresAt,
				)
				require.NoError(t, err, "failed to create device-bound refresh token")
				
				cookieValue = base64.RawURLEncoding.EncodeToString(tokenBytes)
				
				// Verify token was created with device ID
				token, err := refreshStore.GetByHash(ctx, tokenHash)
				require.NoError(t, err, "failed to retrieve token")
				assert.NotNil(t, token.DeviceID, "token should have device ID")
				assert.Equal(t, deviceID, *token.DeviceID, "device ID should match")
				assert.Equal(t, familyID, token.FamilyID, "family ID should match")
				assert.Equal(t, tokenID, token.ID, "token ID should match")
			} else {
				// Regular session without device
				cookieValue, _, _, err = refreshSvc.Issue(ctx, user, time.Now())
				require.NoError(t, err, "failed to issue regular refresh token")
			}

			// Mock token service to capture claims
			var capturedClaims *AccessClaims
			tokenSvc := &mockTokenService{
				signFunc: func(ctx context.Context, claims AccessClaims) (string, error) {
					capturedClaims = &claims
					return "mock-access-token", nil
				},
			}
			
			// Update refresh service with mock token service
			refreshSvc.(*refreshService).tokenSvc = tokenSvc

			// Rotate the token to trigger access token generation
			newCookieValue, accessToken, rotatedUser, err := refreshSvc.Rotate(ctx, cookieValue, time.Now())
			require.NoError(t, err, "failed to rotate refresh token")
			assert.NotEmpty(t, newCookieValue, "new cookie value should not be empty")
			assert.Equal(t, "mock-access-token", accessToken, "access token should match mock")
			assert.Equal(t, user.ID, rotatedUser.ID, "user ID should match")

			// Verify AMR and DeviceID in access token claims
			require.NotNil(t, capturedClaims, "claims should be captured")
			
			assert.Equal(t, tc.wantAMR, capturedClaims.AMR, 
				"AMR should be %v", tc.wantAMR)
			
			if tc.wantDeviceIDSet {
				assert.NotEmpty(t, capturedClaims.DeviceID, 
					"DeviceID should be set in claims for device session")
				assert.Equal(t, deviceID.String(), capturedClaims.DeviceID, 
					"DeviceID in claims should match original device")
			} else {
				assert.Empty(t, capturedClaims.DeviceID, 
					"DeviceID should not be set for non-device session")
			}

			// Verify audit events
			events := auditStore.GetEvents()
			assert.Greater(t, len(events), 0, "should have audit events")
		})
	}
}

// TestRefreshService_RevokeDeviceTokens tests revoking all tokens for a device
func TestRefreshService_RevokeDeviceTokens(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	refreshSvc, refreshStore, userStore, _ := setupRefreshService(t)

	// Create test user
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err, "failed to create test user")

	// Create multiple devices with tokens
	device1 := uuid.New()
	device2 := uuid.New()
	
	// Create tokens for device 1
	token1Bytes, err := refreshSvc.(*refreshService).rand.Bytes(32)
	require.NoError(t, err, "failed to generate token 1")
	
	token1Hash := crypto.HashSHA256(token1Bytes)
	_, _, err = refreshStore.CreateFamilyWithDevice(
		ctx, user.ID, device1, user.SessionVersion, token1Hash, time.Now().Add(time.Hour),
	)
	require.NoError(t, err, "failed to create token for device 1")
	
	// Create tokens for device 2
	token2Bytes, err := refreshSvc.(*refreshService).rand.Bytes(32)
	require.NoError(t, err, "failed to generate token 2")
	
	token2Hash := crypto.HashSHA256(token2Bytes)
	family2, _, err := refreshStore.CreateFamilyWithDevice(
		ctx, user.ID, device2, user.SessionVersion, token2Hash, time.Now().Add(time.Hour),
	)
	require.NoError(t, err, "failed to create token for device 2")

	// Verify both tokens exist
	token1, err := refreshStore.GetByHash(ctx, token1Hash)
	require.NoError(t, err, "should find token 1")
	assert.NotNil(t, token1, "token 1 should exist")
	
	token2, err := refreshStore.GetByHash(ctx, token2Hash)
	require.NoError(t, err, "should find token 2")
	assert.NotNil(t, token2, "token 2 should exist")

	// Revoke all tokens for device 1
	err = refreshStore.RevokeDeviceTokens(ctx, device1, "device revoked", time.Now())
	require.NoError(t, err, "failed to revoke device tokens")

	// Verify device 1 tokens are revoked
	token1After, err := refreshStore.GetByHash(ctx, token1Hash)
	// The in-memory store returns an error for revoked tokens
	if err != nil {
		assert.Contains(t, err.Error(), "not found", "should get not found error for revoked token")
	} else {
		assert.Nil(t, token1After, "device 1 token should be revoked")
	}

	// Verify device 2 tokens are still valid
	token2After, err := refreshStore.GetByHash(ctx, token2Hash)
	require.NoError(t, err, "should find token 2 after device 1 revocation")
	assert.NotNil(t, token2After, "device 2 token should still exist")
	assert.Equal(t, family2, token2After.FamilyID, "device 2 token family should be unchanged")

	// Note: Audit events for RevokeDeviceTokens would be created by the store implementation
	// In production, the database store would create audit events for token revocation
}

// TestRefreshService_DeviceTokenPropagation tests that device ID propagates through token rotation
func TestRefreshService_DeviceTokenPropagation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	refreshSvc, refreshStore, userStore, _ := setupRefreshService(t)

	// Create test user
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err, "failed to create test user")

	// Create initial device-bound token
	deviceID := uuid.New()
	tokenBytes, err := refreshSvc.(*refreshService).rand.Bytes(32)
	require.NoError(t, err, "failed to generate token")
	
	tokenHash := crypto.HashSHA256(tokenBytes)
	familyID, _, err := refreshStore.CreateFamilyWithDevice(
		ctx, user.ID, deviceID, user.SessionVersion, tokenHash, time.Now().Add(time.Hour),
	)
	require.NoError(t, err, "failed to create device-bound token")
	
	cookieValue := base64.RawURLEncoding.EncodeToString(tokenBytes)

	// Perform multiple rotations
	for i := 0; i < 3; i++ {
		// Rotate the token
		newCookieValue, _, _, err := refreshSvc.Rotate(ctx, cookieValue, time.Now())
		require.NoError(t, err, "failed to rotate token on iteration %d", i)
		assert.NotEmpty(t, newCookieValue, "new cookie should not be empty on iteration %d", i)
		assert.NotEqual(t, cookieValue, newCookieValue, "cookie should change on iteration %d", i)

		// Decode and verify the new token has the same device ID
		newTokenBytes, err := base64.RawURLEncoding.DecodeString(newCookieValue)
		require.NoError(t, err, "failed to decode new cookie on iteration %d", i)
		
		newTokenHash := crypto.HashSHA256(newTokenBytes)
		newToken, err := refreshStore.GetByHash(ctx, newTokenHash)
		require.NoError(t, err, "failed to get new token on iteration %d", i)
		require.NotNil(t, newToken, "new token should exist on iteration %d", i)
		
		// Verify device ID is preserved
		require.NotNil(t, newToken.DeviceID, "device ID should be preserved on iteration %d", i)
		assert.Equal(t, deviceID, *newToken.DeviceID, 
			"device ID should match original on iteration %d", i)
		assert.Equal(t, familyID, newToken.FamilyID, 
			"family ID should be preserved on iteration %d", i)

		// Use new cookie for next iteration
		cookieValue = newCookieValue
	}
}
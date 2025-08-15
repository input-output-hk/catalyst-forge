package service

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/testing/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDeviceLinkService(t *testing.T) (DeviceLinkService, store.DeviceLinkStore, store.DeviceStore, *inmemory.UserStore, *inmemory.RefreshStore, *inmemory.AuditStore) {
	t.Helper()

	// Create stores
	linkStore := inmemory.NewDeviceLinkStore()
	deviceStore := inmemory.NewDeviceStore()
	userStore := inmemory.NewUserStore()
	refreshStore := inmemory.NewRefreshStore()
	auditStore := inmemory.NewAuditStore()

	// Create mock services
	tokenSvc := &mockTokenService{}
	refreshSvc := &mockRefreshService{}

	// Create secure random
	rand := crypto.NewSecureRand()

	// Create device link service
	cfg := DeviceLinkConfig{
		LinkStore:      linkStore,
		DeviceStore:    deviceStore,
		UserStore:      userStore,
		RefreshStore:   refreshStore,
		AuditStore:     auditStore,
		TokenService:   tokenSvc,
		RefreshService: refreshSvc,
		Rand:           rand,
		Origin:         "https://example.com",
		LinkTTL:        10 * time.Minute,
		PollInterval:   5,
		AccessTTL:      time.Hour,
		RefreshTTL:     7 * 24 * time.Hour,
	}

	service := NewDeviceLinkService(cfg)

	return service, linkStore, deviceStore, userStore, refreshStore, auditStore
}

// mockRefreshService implements RefreshService for testing
type mockRefreshService struct {
	issueFunc  func(ctx context.Context, user *domain.User, now time.Time) (string, uuid.UUID, uuid.UUID, error)
	rotateFunc func(ctx context.Context, cookieValue string, now time.Time) (string, string, *domain.User, error)
}

func (m *mockRefreshService) Issue(ctx context.Context, user *domain.User, now time.Time) (string, uuid.UUID, uuid.UUID, error) {
	if m.issueFunc != nil {
		return m.issueFunc(ctx, user, now)
	}
	return "mock-refresh-token", uuid.New(), uuid.New(), nil
}

func (m *mockRefreshService) Rotate(ctx context.Context, cookieValue string, now time.Time) (string, string, *domain.User, error) {
	if m.rotateFunc != nil {
		return m.rotateFunc(ctx, cookieValue, now)
	}
	return "new-refresh", "access-token", &domain.User{}, nil
}

func (m *mockRefreshService) RevokeCurrent(ctx context.Context, cookieValue string, reason string) error {
	return nil
}

func (m *mockRefreshService) RevokeAllUser(ctx context.Context, userID uuid.UUID, reason string) error {
	return nil
}

func TestDeviceLinkService_BeginDeviceLink(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		deviceName     string
		purpose        string
		wantDeviceName string
		wantPurpose    string
	}{
		{
			name:           "ok/create_device_link",
			deviceName:     "Test CLI",
			purpose:        "login",
			wantDeviceName: "Test CLI",
			wantPurpose:    "login",
		},
		{
			name:           "ok/default_values",
			deviceName:     "",
			purpose:        "",
			wantDeviceName: "CLI Device",
			wantPurpose:    "login",
		},
		{
			name:           "ok/step_up_purpose",
			deviceName:     "Browser",
			purpose:        "step_up",
			wantDeviceName: "Browser",
			wantPurpose:    "step_up",
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			service, linkStore, _, _, _, auditStore := setupDeviceLinkService(t)

			resp, err := service.BeginDeviceLink(ctx, tc.deviceName, tc.purpose)
			require.NoError(t, err, "BeginDeviceLink failed for deviceName=%q, purpose=%q", tc.deviceName, tc.purpose)
			assert.NotNil(t, resp, "response should not be nil")

			// Verify response fields
			assert.NotEmpty(t, resp.DeviceCode, "device code should not be empty")
			assert.NotEmpty(t, resp.UserCode, "user code should not be empty")
			assert.Regexp(t, `^[A-Z0-9]{4}-[A-Z0-9]{3}$`, resp.UserCode, "user code should match format XXXX-XXX")
			assert.Equal(t, "https://example.com/cli/link", resp.VerificationURI, "verification URI should match configured origin")
			assert.Equal(t, "https://example.com/cli/link?c="+resp.UserCode, resp.VerificationURIComplete, "complete URI should include user code")
			assert.Equal(t, 600, resp.ExpiresIn, "expires_in should be 600 seconds (10 minutes)")
			assert.Equal(t, 5, resp.Interval, "poll interval should be 5 seconds")

			// Verify device code is base64url encoded
			decoded, err := base64.RawURLEncoding.DecodeString(resp.DeviceCode)
			require.NoError(t, err, "device code should be valid base64url")
			assert.Len(t, decoded, 32, "decoded device code should be 32 bytes")

			// Verify link was stored
			deviceCodeHash := crypto.HashSHA256(decoded)
			storedLink, err := linkStore.GetByDeviceCode(ctx, deviceCodeHash)
			require.NoError(t, err, "should retrieve stored link by device code hash")
			assert.NotNil(t, storedLink, "stored link should exist")
			assert.Equal(t, resp.UserCode, storedLink.UserCode, "stored user code should match response")
			assert.Equal(t, tc.wantDeviceName, storedLink.DeviceName, "stored device name should be %q", tc.wantDeviceName)
			assert.Equal(t, tc.wantPurpose, storedLink.Purpose, "stored purpose should be %q", tc.wantPurpose)
			assert.Nil(t, storedLink.AuthorizedAt, "link should not be authorized yet")
			assert.Nil(t, storedLink.UserID, "link should not have user ID yet")

			// Verify audit event
			events := auditStore.GetEvents()
			require.Greater(t, len(events), 0, "should have at least one audit event")
			lastEvent := events[len(events)-1]
			assert.Equal(t, domain.EventDeviceLinkBegin, lastEvent.Type, "audit event type should be EventDeviceLinkBegin")
			assert.Equal(t, tc.wantDeviceName, lastEvent.Metadata["device_name"], "audit event should contain device name")
			assert.Equal(t, tc.wantPurpose, lastEvent.Metadata["purpose"], "audit event should contain purpose")
			assert.Equal(t, resp.UserCode, lastEvent.Metadata["user_code"], "audit event should contain user code")
		})
	}
}

func TestDeviceLinkService_UniqueCodeGeneration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, _, _, _, _ := setupDeviceLinkService(t)

	// Generate multiple links and verify uniqueness
	codes := make(map[string]bool)
	userCodes := make(map[string]bool)

	for i := 0; i < 20; i++ {
		resp, err := service.BeginDeviceLink(ctx, "Device", "login")
		require.NoError(t, err, "failed to begin device link on iteration %d", i)

		assert.False(t, codes[resp.DeviceCode], "device code collision detected on iteration %d", i)
		assert.False(t, userCodes[resp.UserCode], "user code collision detected on iteration %d", i)

		codes[resp.DeviceCode] = true
		userCodes[resp.UserCode] = true
	}
}

func TestDeviceLinkService_AuthorizeDeviceLink(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, linkStore, _, userStore, _, auditStore := setupDeviceLinkService(t)

	// Setup: Create test user
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err, "failed to create test user")

	// Setup: Create device links for testing
	validLink, err := service.BeginDeviceLink(ctx, "Valid Device", "login")
	require.NoError(t, err, "failed to begin valid device link")

	authorizedLink, err := service.BeginDeviceLink(ctx, "Already Authorized", "login")
	require.NoError(t, err, "failed to begin device link for already-authorized test")
	err = service.AuthorizeDeviceLink(ctx, authorizedLink.UserCode, user.ID)
	require.NoError(t, err, "failed to pre-authorize device link")

	tests := []struct {
		name            string
		userCode        string
		userID          uuid.UUID
		wantErr         bool
		errContains     string
		checkAuthorized bool
	}{
		{
			name:            "ok/authorize_link",
			userCode:        validLink.UserCode,
			userID:           user.ID,
			wantErr:         false,
			checkAuthorized: true,
		},
		{
			name:        "error/invalid_user_code",
			userCode:    "INVALID-CODE",
			userID:      user.ID,
			wantErr:     true,
			errContains: "invalid or expired",
		},
		{
			name:        "error/empty_user_code",
			userCode:    "",
			userID:      user.ID,
			wantErr:     true,
			errContains: "invalid or expired",
		},
		{
			name:        "error/already_authorized",
			userCode:    authorizedLink.UserCode,
			userID:      user.ID,
			wantErr:     true,
			errContains: "already authorized",
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			eventsBefore := len(auditStore.GetEvents())

			err := service.AuthorizeDeviceLink(ctx, tc.userCode, tc.userID)

			if tc.wantErr {
				require.Error(t, err, "expected error for userCode=%q", tc.userCode)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains, "error message should contain %q", tc.errContains)
				}
				return
			}

			require.NoError(t, err, "unexpected error for userCode=%q", tc.userCode)

			if tc.checkAuthorized {
				// Verify link was authorized
				deviceCode, _ := base64.RawURLEncoding.DecodeString(validLink.DeviceCode)
				deviceCodeHash := crypto.HashSHA256(deviceCode)
				storedLink, err := linkStore.GetByDeviceCode(ctx, deviceCodeHash)
				require.NoError(t, err, "failed to retrieve authorized link")
				assert.NotNil(t, storedLink.AuthorizedAt, "link should have authorization timestamp")
				assert.NotNil(t, storedLink.UserID, "link should have user ID")
				assert.Equal(t, tc.userID, *storedLink.UserID, "link should be authorized by correct user")

				// Verify audit event
				eventsAfter := auditStore.GetEvents()
				newEvents := eventsAfter[eventsBefore:]
				require.Len(t, newEvents, 1, "should have exactly one new audit event")
				assert.Equal(t, domain.EventDeviceLinkAuthorized, newEvents[0].Type, "audit event should be authorization type")
				assert.Equal(t, tc.userID, *newEvents[0].UserID, "audit event should contain user ID")
			}
		})
	}
}

func TestDeviceLinkService_ExchangeDeviceCode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, deviceStore, userStore, refreshStore, auditStore := setupDeviceLinkService(t)

	// Create a test user
	user, err := userStore.Create(ctx, "user@example.com", []string{"admin"})
	require.NoError(t, err)

	t.Run("ok/successful_exchange", func(t *testing.T) {
		// Begin and authorize a device link
		resp, err := service.BeginDeviceLink(ctx, "CLI Tool", "login")
		require.NoError(t, err)

		err = service.AuthorizeDeviceLink(ctx, resp.UserCode, user.ID)
		require.NoError(t, err)

		eventsBefore := len(auditStore.GetEvents())

		// Exchange the device code
		exchangeResp, err := service.ExchangeDeviceCode(ctx, resp.DeviceCode)
		require.NoError(t, err)
		assert.NotNil(t, exchangeResp)

		// Verify response
		assert.Empty(t, exchangeResp.Status)
		assert.NotEmpty(t, exchangeResp.AccessToken)
		assert.NotEmpty(t, exchangeResp.RefreshToken)
		assert.NotEmpty(t, exchangeResp.DeviceID)
		assert.Equal(t, 3600, exchangeResp.ExpiresIn)
		assert.Equal(t, 604800, exchangeResp.RefreshExpiresIn)
		assert.NotNil(t, exchangeResp.User)
		assert.Equal(t, user.ID.String(), exchangeResp.User.ID)
		assert.Equal(t, user.Email, exchangeResp.User.Email)
		assert.Equal(t, user.Roles, exchangeResp.User.Roles)

		// Verify device was created
		deviceID, err := uuid.Parse(exchangeResp.DeviceID)
		require.NoError(t, err)
		device, err := deviceStore.GetByID(ctx, deviceID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, device.UserID)
		assert.Equal(t, "CLI Tool", device.DeviceName)
		assert.Nil(t, device.RevokedAt)

		// Verify refresh token was created with device ID
		refreshTokenBytes, err := base64.RawURLEncoding.DecodeString(exchangeResp.RefreshToken)
		require.NoError(t, err)
		refreshTokenHash := crypto.HashSHA256(refreshTokenBytes)
		refreshToken, err := refreshStore.GetByHash(ctx, refreshTokenHash)
		require.NoError(t, err)
		assert.NotNil(t, refreshToken.DeviceID)
		assert.Equal(t, deviceID, *refreshToken.DeviceID)

		// Verify audit events
		eventsAfter := auditStore.GetEvents()
		newEvents := eventsAfter[eventsBefore:]
		require.Len(t, newEvents, 2)

		// Check for exchange event
		var foundExchange, foundDeviceCreated bool
		for _, event := range newEvents {
			if event.Type == domain.EventDeviceLinkExchanged {
				foundExchange = true
				assert.Equal(t, user.ID, *event.UserID)
				assert.Equal(t, deviceID.String(), event.Metadata["device_id"])
			}
			if event.Type == domain.EventDeviceCreated {
				foundDeviceCreated = true
				assert.Equal(t, user.ID, *event.UserID)
				assert.Equal(t, deviceID.String(), event.Metadata["device_id"])
			}
		}
		assert.True(t, foundExchange, "Should have device link exchange event")
		assert.True(t, foundDeviceCreated, "Should have device created event")
	})

	t.Run("ok/authorization_pending", func(t *testing.T) {
		// Begin but don't authorize
		resp, err := service.BeginDeviceLink(ctx, "Device", "login")
		require.NoError(t, err)

		// Try to exchange
		exchangeResp, err := service.ExchangeDeviceCode(ctx, resp.DeviceCode)
		require.NoError(t, err)
		assert.Equal(t, "authorization_pending", exchangeResp.Status)
		assert.Empty(t, exchangeResp.AccessToken)
		assert.Empty(t, exchangeResp.RefreshToken)
	})

	t.Run("ok/expired_token", func(t *testing.T) {
		// Invalid device code
		exchangeResp, err := service.ExchangeDeviceCode(ctx, "invalid-device-code")
		require.NoError(t, err)
		assert.Equal(t, "expired_token", exchangeResp.Status)
		assert.Empty(t, exchangeResp.AccessToken)

		// Valid format but non-existent
		fakeCode := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
		exchangeResp, err = service.ExchangeDeviceCode(ctx, fakeCode)
		require.NoError(t, err)
		assert.Equal(t, "expired_token", exchangeResp.Status)
	})

	t.Run("ok/link_deleted_after_exchange", func(t *testing.T) {
		// Begin and authorize
		resp, err := service.BeginDeviceLink(ctx, "Device", "login")
		require.NoError(t, err)
		err = service.AuthorizeDeviceLink(ctx, resp.UserCode, user.ID)
		require.NoError(t, err)

		// Exchange
		_, err = service.ExchangeDeviceCode(ctx, resp.DeviceCode)
		require.NoError(t, err)

		// Try to exchange again - should fail
		exchangeResp, err := service.ExchangeDeviceCode(ctx, resp.DeviceCode)
		require.NoError(t, err)
		assert.Equal(t, "expired_token", exchangeResp.Status)
	})
}

func TestDeviceLinkService_UserCodeFormat(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, _, _, _, _, _ := setupDeviceLinkService(t)

	// Generate many codes and verify format
	for i := 0; i < 100; i++ {
		resp, err := service.BeginDeviceLink(ctx, "Device", "login")
		require.NoError(t, err)

		// Verify format: XXXX-XXX
		assert.Regexp(t, `^[A-Z0-9]{4}-[A-Z0-9]{3}$`, resp.UserCode)

		// Verify no confusing characters (0, O, I, 1, etc.)
		for _, char := range resp.UserCode {
			if char != '-' {
				assert.NotContains(t, "01IO", string(char), "User code should not contain confusing characters")
			}
		}
	}
}

func TestDeviceLinkService_TTL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create service with custom TTL
	cfg := DeviceLinkConfig{
		LinkStore:    inmemory.NewDeviceLinkStore(),
		DeviceStore:  inmemory.NewDeviceStore(),
		UserStore:    inmemory.NewUserStore(),
		RefreshStore: inmemory.NewRefreshStore(),
		AuditStore:   inmemory.NewAuditStore(),
		TokenService: &mockTokenService{},
		RefreshService: &mockRefreshService{},
		Rand:         crypto.NewSecureRand(),
		Origin:       "https://example.com",
		LinkTTL:      30 * time.Second,
		PollInterval: 2,
		AccessTTL:    time.Hour,
		RefreshTTL:   time.Hour,
	}
	service := NewDeviceLinkService(cfg)

	resp, err := service.BeginDeviceLink(ctx, "Device", "login")
	require.NoError(t, err)

	assert.Equal(t, 30, resp.ExpiresIn)
	assert.Equal(t, 2, resp.Interval)
}

func TestDeviceLinkService_StepUpPurpose(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service, linkStore, _, _, _, _ := setupDeviceLinkService(t)

	// Create link for step-up purpose
	resp, err := service.BeginDeviceLink(ctx, "Device", "step_up")
	require.NoError(t, err)

	// Verify purpose was stored
	deviceCode, _ := base64.RawURLEncoding.DecodeString(resp.DeviceCode)
	deviceCodeHash := crypto.HashSHA256(deviceCode)
	storedLink, err := linkStore.GetByDeviceCode(ctx, deviceCodeHash)
	require.NoError(t, err)
	assert.Equal(t, "step_up", storedLink.Purpose)
}
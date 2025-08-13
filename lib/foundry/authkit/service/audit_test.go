package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/service"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/testing/inmemory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLogger_LogUserCreated(t *testing.T) {
	ctx := context.Background()
	store := inmemory.NewAuditStore()
	logger := service.NewAuditLogger(store)
	
	userID := uuid.New()
	email := "user@example.com"
	roles := []string{"user", "admin"}
	
	err := logger.LogUserCreated(ctx, userID, email, roles)
	require.NoError(t, err)
	
	// Verify event was stored
	events := store.GetEvents()
	require.Len(t, events, 1)
	
	event := events[0]
	assert.Equal(t, domain.EventUserCreated, event.Type)
	assert.Equal(t, userID, *event.UserID)
	assert.Equal(t, userID, *event.ActorID)
	
	// Verify metadata
	metadata := event.Metadata
	assert.Equal(t, "example.com", metadata["email_domain"])
	assert.Equal(t, roles, metadata["roles"])
	
	// Ensure no PII (full email) is stored
	assert.NotContains(t, metadata, "email")
}

func TestAuditLogger_LogLoginAttempt(t *testing.T) {
	tests := []struct {
		name      string
		userID    *uuid.UUID
		success   bool
		reason    string
		wantEvent domain.EventType
	}{
		{
			name:      "successful login",
			userID:    ptr(uuid.New()),
			success:   true,
			reason:    "",
			wantEvent: domain.EventLoginSuccess,
		},
		{
			name:      "failed login with reason",
			userID:    ptr(uuid.New()),
			success:   false,
			reason:    "invalid_credentials",
			wantEvent: domain.EventLoginFailed,
		},
		{
			name:      "failed login without user",
			userID:    nil,
			success:   false,
			reason:    "user_not_found",
			wantEvent: domain.EventLoginFailed,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := inmemory.NewAuditStore()
			logger := service.NewAuditLogger(store)
			
			err := logger.LogLoginAttempt(ctx, tt.userID, tt.success, tt.reason)
			require.NoError(t, err)
			
			events := store.GetEvents()
			require.Len(t, events, 1)
			
			event := events[0]
			assert.Equal(t, tt.wantEvent, event.Type)
			assert.Equal(t, tt.userID, event.UserID)
			assert.Equal(t, tt.success, event.Metadata["success"])
			
			if tt.reason != "" && !tt.success {
				assert.Equal(t, tt.reason, event.Metadata["reason"])
			}
		})
	}
}

func TestAuditLogger_LogCredentialUsed(t *testing.T) {
	tests := []struct {
		name               string
		credentialID       []byte
		expectedHexLength  int
	}{
		{
			name:              "normal credential ID",
			credentialID:      []byte("credential-id-123456789012345678"),
			expectedHexLength: 16, // 8 bytes as hex
		},
		{
			name:              "short credential ID",
			credentialID:      []byte("short"),
			expectedHexLength: 10, // 5 bytes as hex
		},
		{
			name:              "empty credential ID",
			credentialID:      []byte{},
			expectedHexLength: 0,
		},
		{
			name:              "exactly 8 bytes",
			credentialID:      []byte("12345678"),
			expectedHexLength: 16, // 8 bytes as hex
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := inmemory.NewAuditStore()
			logger := service.NewAuditLogger(store)
			
			userID := uuid.New()
			deviceName := "MacBook Pro - Chrome"
			
			err := logger.LogCredentialUsed(ctx, userID, tt.credentialID, deviceName)
			require.NoError(t, err)
			
			events := store.GetEvents()
			require.Len(t, events, 1)
			
			event := events[0]
			assert.Equal(t, domain.EventCredentialUsed, event.Type)
			assert.Equal(t, userID, *event.UserID)
			
			// Verify only partial credential ID is stored (privacy)
			metadata := event.Metadata
			credID := metadata["credential_id"].(string)
			assert.Len(t, credID, tt.expectedHexLength)
			if len(tt.credentialID) > 0 {
				assert.NotEqual(t, string(tt.credentialID), credID)
			}
			assert.Equal(t, deviceName, metadata["device_name"])
		})
	}
}

func TestAuditLogger_LogTokenReplay(t *testing.T) {
	ctx := context.Background()
	store := inmemory.NewAuditStore()
	logger := service.NewAuditLogger(store)
	
	userID := uuid.New()
	familyID := uuid.New()
	
	err := logger.LogTokenReplay(ctx, userID, familyID)
	require.NoError(t, err)
	
	events := store.GetEvents()
	require.Len(t, events, 1)
	
	event := events[0]
	assert.Equal(t, domain.EventTokenReplayDetected, event.Type)
	assert.Equal(t, userID, *event.UserID)
	assert.Equal(t, familyID.String(), event.Metadata["family_id"])
	assert.Equal(t, "family_revoked", event.Metadata["action"])
}

func TestAuditLogger_LogWebAuthnAnomaly(t *testing.T) {
	tests := []struct {
		name        string
		anomalyType string
		wantEvent   domain.EventType
	}{
		{
			name:        "sign count anomaly",
			anomalyType: "signcount",
			wantEvent:   domain.EventWebAuthnSignCountAnomaly,
		},
		{
			name:        "origin mismatch",
			anomalyType: "origin",
			wantEvent:   domain.EventWebAuthnOriginMismatch,
		},
		{
			name:        "challenge failed",
			anomalyType: "challenge",
			wantEvent:   domain.EventWebAuthnChallengeFailed,
		},
		{
			name:        "attestation failed",
			anomalyType: "attestation",
			wantEvent:   domain.EventWebAuthnAttestationFailed,
		},
		{
			name:        "unknown anomaly",
			anomalyType: "unknown",
			wantEvent:   domain.EventSuspiciousActivity,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := inmemory.NewAuditStore()
			logger := service.NewAuditLogger(store)
			
			userID := uuid.New()
			details := map[string]interface{}{
				"expected": "value1",
				"actual":   "value2",
				"password": "should-not-be-logged", // Sensitive field
			}
			
			err := logger.LogWebAuthnAnomaly(ctx, userID, tt.anomalyType, details)
			require.NoError(t, err)
			
			events := store.GetEvents()
			require.Len(t, events, 1)
			
			event := events[0]
			assert.Equal(t, tt.wantEvent, event.Type)
			assert.Equal(t, userID, *event.UserID)
			
			// Verify sensitive data is not logged
			metadata := event.Metadata
			assert.Equal(t, tt.anomalyType, metadata["anomaly_type"])
			assert.Equal(t, "value1", metadata["expected"])
			assert.Equal(t, "value2", metadata["actual"])
			assert.NotContains(t, metadata, "password")
		})
	}
}

func TestAuditLogger_LogInviteFailed(t *testing.T) {
	tests := []struct {
		name      string
		attempts  int
		wantEvent domain.EventType
		wantLocked bool
	}{
		{
			name:      "first attempt",
			attempts:  1,
			wantEvent: domain.EventInviteFailed,
			wantLocked: false,
		},
		{
			name:      "fourth attempt",
			attempts:  4,
			wantEvent: domain.EventInviteFailed,
			wantLocked: false,
		},
		{
			name:      "fifth attempt - locked",
			attempts:  5,
			wantEvent: domain.EventInviteLocked,
			wantLocked: true,
		},
		{
			name:      "beyond limit",
			attempts:  10,
			wantEvent: domain.EventInviteLocked,
			wantLocked: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := inmemory.NewAuditStore()
			logger := service.NewAuditLogger(store)
			
			inviteID := uuid.New()
			reason := "invalid_token"
			
			err := logger.LogInviteFailed(ctx, inviteID, reason, tt.attempts)
			require.NoError(t, err)
			
			events := store.GetEvents()
			require.Len(t, events, 1)
			
			event := events[0]
			assert.Equal(t, tt.wantEvent, event.Type)
			assert.Nil(t, event.UserID)
			
			metadata := event.Metadata
			assert.Equal(t, inviteID.String(), metadata["invite_id"])
			assert.Equal(t, reason, metadata["reason"])
			assert.Equal(t, tt.attempts, metadata["attempts"])
			
			if tt.wantLocked {
				assert.Equal(t, true, metadata["locked"])
			} else {
				assert.NotContains(t, metadata, "locked")
			}
		})
	}
}

func TestAuditLogger_LogAdminAction(t *testing.T) {
	tests := []struct {
		name      string
		action    string
		wantEvent domain.EventType
	}{
		{
			name:      "force logout",
			action:    "force_logout",
			wantEvent: domain.EventAdminForceLogout,
		},
		{
			name:      "modify user",
			action:    "modify_user",
			wantEvent: domain.EventAdminUserModified,
		},
		{
			name:      "revoke credential",
			action:    "revoke_credential",
			wantEvent: domain.EventAdminCredentialRevoked,
		},
		{
			name:      "generic action",
			action:    "custom_action",
			wantEvent: domain.EventAdminAction,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := inmemory.NewAuditStore()
			logger := service.NewAuditLogger(store)
			
			actorID := uuid.New()
			targetUserID := uuid.New()
			details := map[string]interface{}{
				"reason":    "security_violation",
				"secret":    "should-not-log", // Sensitive
				"timestamp": time.Now().Unix(),
			}
			
			err := logger.LogAdminAction(ctx, actorID, tt.action, &targetUserID, details)
			require.NoError(t, err)
			
			events := store.GetEvents()
			require.Len(t, events, 1)
			
			event := events[0]
			assert.Equal(t, tt.wantEvent, event.Type)
			assert.Equal(t, targetUserID, *event.UserID)
			assert.Equal(t, actorID, *event.ActorID)
			
			// Verify metadata
			metadata := event.Metadata
			assert.Equal(t, tt.action, metadata["action"])
			assert.Equal(t, targetUserID.String(), metadata["target_user_id"])
			assert.Equal(t, "security_violation", metadata["reason"])
			assert.Contains(t, metadata, "timestamp")
			
			// Ensure sensitive field is not logged
			assert.NotContains(t, metadata, "secret")
		})
	}
}

func TestAuditLogger_NoPIILogged(t *testing.T) {
	// Test that sensitive fields are never logged in metadata
	ctx := context.Background()
	store := inmemory.NewAuditStore()
	logger := service.NewAuditLogger(store)
	
	// Test various sensitive field names
	sensitiveData := map[string]interface{}{
		"password":          "secret123",
		"token":            "jwt-token-here",
		"secret":           "api-secret",
		"private_key":      "-----BEGIN RSA PRIVATE KEY-----",
		"clientDataJSON":   "webauthn-data",
		"authenticatorData": "more-webauthn",
		"recovery_code":    "ABC123",
		"hash":            "sha256:abcdef",
		"Token":           "uppercase-token",
		"PASSWORD":        "uppercase-password",
	}
	
	// Log with sensitive data
	userID := uuid.New()
	err := logger.LogWebAuthnAnomaly(ctx, userID, "test", sensitiveData)
	require.NoError(t, err)
	
	events := store.GetEvents()
	require.Len(t, events, 1)
	
	// Verify none of the sensitive fields are in metadata
	metadata := events[0].Metadata
	for key := range sensitiveData {
		assert.NotContains(t, metadata, key, "Sensitive field %s should not be logged", key)
	}
}

func TestAuditLogger_LogTokenRefresh(t *testing.T) {
	tests := []struct {
		name      string
		success   bool
		wantEvent domain.EventType
	}{
		{
			name:      "successful refresh",
			success:   true,
			wantEvent: domain.EventTokenRefresh,
		},
		{
			name:      "failed refresh",
			success:   false,
			wantEvent: domain.EventTokenRefreshFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := inmemory.NewAuditStore()
			logger := service.NewAuditLogger(store)

			userID := uuid.New()
			familyID := uuid.New()

			err := logger.LogTokenRefresh(ctx, userID, familyID, tt.success)
			require.NoError(t, err)

			events := store.GetEvents()
			require.Len(t, events, 1)

			event := events[0]
			assert.Equal(t, tt.wantEvent, event.Type)
			assert.Equal(t, userID, *event.UserID)
			assert.Equal(t, userID, *event.ActorID)

			metadata := event.Metadata
			assert.Equal(t, familyID.String(), metadata["family_id"])
			assert.Equal(t, tt.success, metadata["success"])
		})
	}
}

func TestAuditLogger_LogRecoveryAttempt(t *testing.T) {
	tests := []struct {
		name      string
		success   bool
		wantEvent domain.EventType
	}{
		{
			name:      "successful recovery",
			success:   true,
			wantEvent: domain.EventRecoverySuccess,
		},
		{
			name:      "failed recovery",
			success:   false,
			wantEvent: domain.EventRecoveryFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := inmemory.NewAuditStore()
			logger := service.NewAuditLogger(store)

			userID := uuid.New()

			err := logger.LogRecoveryAttempt(ctx, userID, tt.success)
			require.NoError(t, err)

			events := store.GetEvents()
			require.Len(t, events, 1)

			event := events[0]
			assert.Equal(t, tt.wantEvent, event.Type)
			assert.Equal(t, userID, *event.UserID)
			assert.Equal(t, userID, *event.ActorID)
			assert.Equal(t, tt.success, event.Metadata["success"])
		})
	}
}

func TestAuditLogger_LogStepUp(t *testing.T) {
	tests := []struct {
		name      string
		success   bool
		reason    string
		wantEvent domain.EventType
	}{
		{
			name:      "successful step-up",
			success:   true,
			reason:    "admin_action_required",
			wantEvent: domain.EventStepUpSuccess,
		},
		{
			name:      "failed step-up",
			success:   false,
			reason:    "insufficient_auth",
			wantEvent: domain.EventStepUpFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := inmemory.NewAuditStore()
			logger := service.NewAuditLogger(store)

			userID := uuid.New()

			err := logger.LogStepUp(ctx, userID, tt.success, tt.reason)
			require.NoError(t, err)

			events := store.GetEvents()
			require.Len(t, events, 1)

			event := events[0]
			assert.Equal(t, tt.wantEvent, event.Type)
			assert.Equal(t, userID, *event.UserID)
			assert.Equal(t, userID, *event.ActorID)

			metadata := event.Metadata
			assert.Equal(t, tt.success, metadata["success"])
			assert.Equal(t, tt.reason, metadata["reason"])
		})
	}
}

func TestAuditLogger_LogInviteCreated(t *testing.T) {
	ctx := context.Background()
	store := inmemory.NewAuditStore()
	logger := service.NewAuditLogger(store)

	actorID := uuid.New()
	inviteID := uuid.New()
	email := "newuser@company.com"
	roles := []string{"user", "viewer"}

	err := logger.LogInviteCreated(ctx, actorID, inviteID, email, roles)
	require.NoError(t, err)

	events := store.GetEvents()
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, domain.EventInviteCreated, event.Type)
	assert.Nil(t, event.UserID) // No user yet
	assert.Equal(t, actorID, *event.ActorID)

	metadata := event.Metadata
	assert.Equal(t, inviteID.String(), metadata["invite_id"])
	assert.Equal(t, "company.com", metadata["email_domain"])
	assert.Equal(t, roles, metadata["roles"])

	// Ensure full email is not logged
	assert.NotContains(t, metadata, "email")
}

func TestAuditLogger_LogInviteRedeemed(t *testing.T) {
	ctx := context.Background()
	store := inmemory.NewAuditStore()
	logger := service.NewAuditLogger(store)

	userID := uuid.New()
	inviteID := uuid.New()

	err := logger.LogInviteRedeemed(ctx, userID, inviteID)
	require.NoError(t, err)

	events := store.GetEvents()
	require.Len(t, events, 1)

	event := events[0]
	assert.Equal(t, domain.EventInviteRedeemed, event.Type)
	assert.Equal(t, userID, *event.UserID)
	assert.Equal(t, userID, *event.ActorID)
	assert.Equal(t, inviteID.String(), event.Metadata["invite_id"])
}

func TestAuditLogger_LogSessionInvalidated(t *testing.T) {
	tests := []struct {
		name        string
		reason      string
		actorID     *uuid.UUID
		userID      uuid.UUID
		wantEvent   domain.EventType
		description string
	}{
		{
			name:        "user logout",
			reason:      "user_requested",
			actorID:     nil,
			userID:      uuid.New(),
			wantEvent:   domain.EventLogoutAll,
			description: "User-initiated logout",
		},
		{
			name:        "admin forced logout - different actor",
			reason:      "forced",
			actorID:     ptr(uuid.New()),
			userID:      uuid.New(),
			wantEvent:   domain.EventLogoutForced,
			description: "Admin forced logout with different actor",
		},
		{
			name:        "system logout",
			reason:      "security_violation",
			actorID:     nil,
			userID:      uuid.New(),
			wantEvent:   domain.EventLogoutAll,
			description: "System-initiated logout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := inmemory.NewAuditStore()
			logger := service.NewAuditLogger(store)

			// For self-forced logout test, make actorID same as userID
			if tt.name == "admin forced logout - different actor" {
				// Keep different actorID
			} else if tt.reason == "forced" && tt.actorID != nil {
				tt.actorID = &tt.userID // Make them the same for self-forced
			}

			err := logger.LogSessionInvalidated(ctx, tt.userID, tt.reason, tt.actorID)
			require.NoError(t, err)

			events := store.GetEvents()
			require.Len(t, events, 1)

			event := events[0]
			assert.Equal(t, tt.wantEvent, event.Type)
			assert.Equal(t, tt.userID, *event.UserID)

			if tt.actorID != nil {
				assert.Equal(t, *tt.actorID, *event.ActorID)
			} else {
				assert.Nil(t, event.ActorID)
			}

			metadata := event.Metadata
			assert.Equal(t, tt.reason, metadata["reason"])
		})
	}
}

func TestAuditLogger_ConcurrentWrites(t *testing.T) {
	ctx := context.Background()
	store := inmemory.NewAuditStore()
	logger := service.NewAuditLogger(store)

	// Test concurrent audit logging
	numGoroutines := 50
	eventsPerGoroutine := 20
	totalEvents := numGoroutines * eventsPerGoroutine

	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			var err error
			for j := 0; j < eventsPerGoroutine; j++ {
				userID := uuid.New()
				switch j % 5 {
				case 0:
					err = logger.LogLoginAttempt(ctx, &userID, true, "")
				case 1:
					err = logger.LogCredentialUsed(ctx, userID, []byte("cred123"), "Device")
				case 2:
					err = logger.LogTokenRefresh(ctx, userID, uuid.New(), true)
				case 3:
					err = logger.LogRecoveryAttempt(ctx, userID, true)
				case 4:
					err = logger.LogStepUp(ctx, userID, true, "admin")
				}
				
				if err != nil {
					done <- err
					return
				}
			}
			done <- nil
		}(i)
	}

	// Wait for all goroutines
	var errors []error
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			errors = append(errors, err)
		}
	}

	// Should have no errors
	assert.Empty(t, errors, "Expected no errors from concurrent writes")

	// Verify all events were logged
	events := store.GetEvents()
	assert.Len(t, events, totalEvents, "All events should be logged")

	// Verify event types distribution
	eventCounts := make(map[domain.EventType]int)
	for _, event := range events {
		eventCounts[event.Type]++
	}

	expectedPerType := totalEvents / 5
	assert.Equal(t, expectedPerType, eventCounts[domain.EventLoginSuccess])
	assert.Equal(t, expectedPerType, eventCounts[domain.EventCredentialUsed])
	assert.Equal(t, expectedPerType, eventCounts[domain.EventTokenRefresh])
	assert.Equal(t, expectedPerType, eventCounts[domain.EventRecoverySuccess])
	assert.Equal(t, expectedPerType, eventCounts[domain.EventStepUpSuccess])
}

func TestAuditLogger_EventStructureIntegrity(t *testing.T) {
	ctx := context.Background()
	store := inmemory.NewAuditStore()
	logger := service.NewAuditLogger(store)

	userID := uuid.New()
	actorID := uuid.New()
	metadata := map[string]interface{}{
		"test_field": "test_value",
		"number":     42,
		"boolean":    true,
	}

	err := logger.LogEvent(ctx, domain.EventUserCreated, &userID, &actorID, metadata)
	require.NoError(t, err)

	events := store.GetEvents()
	require.Len(t, events, 1)

	event := events[0]

	// Verify all required fields are set
	assert.NotEqual(t, uuid.Nil, event.ID)
	assert.Equal(t, domain.EventUserCreated, event.Type)
	assert.Equal(t, userID, *event.UserID)
	assert.Equal(t, actorID, *event.ActorID)
	assert.NotZero(t, event.CreatedAt)

	// Verify metadata integrity
	assert.Equal(t, "test_value", event.Metadata["test_field"])
	assert.Equal(t, 42, event.Metadata["number"])
	assert.Equal(t, true, event.Metadata["boolean"])

	// Verify timestamp is recent
	assert.WithinDuration(t, time.Now().UTC(), event.CreatedAt, time.Second)
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getDomain", func(t *testing.T) {
		tests := []struct {
			email    string
			expected string
		}{
			{"user@example.com", "example.com"},
			{"USER@EXAMPLE.COM", "example.com"}, // Case normalization
			{"  user@example.com  ", "example.com"}, // Whitespace handling
			{"user@sub.example.com", "sub.example.com"},
			{"invalid-email", "unknown"},
			{"@example.com", "example.com"},
			{"user@", "unknown"},
			{"", "unknown"},
		}

		for _, tt := range tests {
			t.Run(tt.email, func(t *testing.T) {
				// We need to test the getDomain function indirectly through LogUserCreated
				ctx := context.Background()
				store := inmemory.NewAuditStore()
				logger := service.NewAuditLogger(store)

				userID := uuid.New()
				err := logger.LogUserCreated(ctx, userID, tt.email, []string{"user"})
				require.NoError(t, err)

				events := store.GetEvents()
				require.Len(t, events, 1)

				domain := events[0].Metadata["email_domain"]
				assert.Equal(t, tt.expected, domain)
			})
		}
	})

	t.Run("isSensitiveField", func(t *testing.T) {
		// Test through LogWebAuthnAnomaly which uses the function
		ctx := context.Background()
		store := inmemory.NewAuditStore()
		logger := service.NewAuditLogger(store)

		userID := uuid.New()
		testData := map[string]interface{}{
			"safe_field":        "should_appear",
			"password":          "should_not_appear",
			"SECRET_KEY":        "should_not_appear",
			"token123":          "should_not_appear",
			"user_signature":    "should_not_appear",
			"private_data":      "should_not_appear",
			"clientDataJSON":    "should_not_appear",
			"authenticatorData": "should_not_appear",
			"recovery_code":     "should_not_appear",
			"sha256_hash":       "should_not_appear",
			"normal_value":      "should_appear",
		}

		err := logger.LogWebAuthnAnomaly(ctx, userID, "test", testData)
		require.NoError(t, err)

		events := store.GetEvents()
		require.Len(t, events, 1)

		metadata := events[0].Metadata

		// Verify safe fields appear
		assert.Equal(t, "should_appear", metadata["safe_field"])
		assert.Equal(t, "should_appear", metadata["normal_value"])

		// Verify sensitive fields are filtered out
		assert.NotContains(t, metadata, "password")
		assert.NotContains(t, metadata, "SECRET_KEY")
		assert.NotContains(t, metadata, "token123")
		assert.NotContains(t, metadata, "user_signature")
		assert.NotContains(t, metadata, "private_data")
		assert.NotContains(t, metadata, "clientDataJSON")
		assert.NotContains(t, metadata, "authenticatorData")
		assert.NotContains(t, metadata, "recovery_code")
		assert.NotContains(t, metadata, "sha256_hash")
	})
}

func TestAuditLogger_ErrorHandling(t *testing.T) {
	// Test with a failing store
	failingStore := &failingAuditStore{shouldFail: true}
	logger := service.NewAuditLogger(failingStore)

	ctx := context.Background()
	userID := uuid.New()

	// Should return the error from the store
	err := logger.LogLoginAttempt(ctx, &userID, true, "")
	assert.Error(t, err)
}

func TestAuditLogger_MetadataIsolation(t *testing.T) {
	// Test that metadata modifications don't affect stored events
	ctx := context.Background()
	store := inmemory.NewAuditStore()
	logger := service.NewAuditLogger(store)

	userID := uuid.New()
	originalMetadata := map[string]interface{}{
		"mutable_field": "original_value",
	}

	err := logger.LogEvent(ctx, domain.EventUserCreated, &userID, &userID, originalMetadata)
	require.NoError(t, err)

	// Modify the original metadata
	originalMetadata["mutable_field"] = "modified_value"
	originalMetadata["new_field"] = "should_not_appear"

	// Verify stored event wasn't affected
	events := store.GetEvents()
	require.Len(t, events, 1)

	storedMetadata := events[0].Metadata
	assert.Equal(t, "original_value", storedMetadata["mutable_field"])
	assert.NotContains(t, storedMetadata, "new_field")
}

// Mock failing audit store for error testing
type failingAuditStore struct {
	shouldFail bool
}

func (f *failingAuditStore) Record(ctx context.Context, event domain.Event) error {
	if f.shouldFail {
		return assert.AnError
	}
	return nil
}

// Helper function to create pointer
func ptr[T any](v T) *T {
	return &v
}
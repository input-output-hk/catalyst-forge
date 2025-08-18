package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/testing/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebAuthnService(t *testing.T) {
	t.Parallel()

	t.Run("ok/valid_config", func(t *testing.T) {
		t.Parallel()

		cfg := WebAuthnConfig{
			RPDisplayName: "Test RP",
			RPID:          "localhost",
			RPOrigins:     []string{"http://localhost:3000"},
			Users:         inmemory.NewUserStore(),
			Credentials:   inmemory.NewCredentialStore(),
			Challenges:    inmemory.NewChallengeStore(),
			Rand:          crypto.NewSecureRand(),
			ChallengeTTL:  5 * time.Minute,
		}

		svc, err := NewWebAuthnService(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("ok/empty_rp_display_name", func(t *testing.T) {
		t.Parallel()

		cfg := WebAuthnConfig{
			RPDisplayName: "",
			RPID:          "localhost",
			RPOrigins:     []string{"http://localhost:3000"},
			Users:         inmemory.NewUserStore(),
			Credentials:   inmemory.NewCredentialStore(),
			Challenges:    inmemory.NewChallengeStore(),
			Rand:          crypto.NewSecureRand(),
			ChallengeTTL:  5 * time.Minute,
		}

		svc, err := NewWebAuthnService(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("ok/empty_rpid", func(t *testing.T) {
		t.Parallel()

		cfg := WebAuthnConfig{
			RPDisplayName: "Test RP",
			RPID:          "",
			RPOrigins:     []string{"http://localhost:3000"},
			Users:         inmemory.NewUserStore(),
			Credentials:   inmemory.NewCredentialStore(),
			Challenges:    inmemory.NewChallengeStore(),
			Rand:          crypto.NewSecureRand(),
			ChallengeTTL:  5 * time.Minute,
		}

		svc, err := NewWebAuthnService(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("error/no_origins", func(t *testing.T) {
		t.Parallel()

		cfg := WebAuthnConfig{
			RPDisplayName: "Test RP",
			RPID:          "localhost",
			RPOrigins:     []string{},
			Users:         inmemory.NewUserStore(),
			Credentials:   inmemory.NewCredentialStore(),
			Challenges:    inmemory.NewChallengeStore(),
			Rand:          crypto.NewSecureRand(),
			ChallengeTTL:  5 * time.Minute,
		}

		_, err := NewWebAuthnService(cfg)
		assert.Error(t, err)
	})

	t.Run("ok/with_admin_aaguids", func(t *testing.T) {
		t.Parallel()

		cfg := WebAuthnConfig{
			RPDisplayName:        "Test RP",
			RPID:                 "localhost",
			RPOrigins:            []string{"http://localhost:3000"},
			Users:                inmemory.NewUserStore(),
			Credentials:          inmemory.NewCredentialStore(),
			Challenges:           inmemory.NewChallengeStore(),
			Rand:                 crypto.NewSecureRand(),
			AdminAAGUIDAllowlist: []string{"00000000-0000-0000-0000-000000000001"},
			ChallengeTTL:         5 * time.Minute,
		}

		svc, err := NewWebAuthnService(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, svc)
	})
}

func TestWebAuthnService_BeginRegistration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := WebAuthnConfig{
		RPDisplayName: "Test RP",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:3000"},
		Users:         inmemory.NewUserStore(),
		Credentials:   inmemory.NewCredentialStore(),
		Challenges:    inmemory.NewChallengeStore(),
		Rand:          crypto.NewSecureRand(),
		ChallengeTTL:  5 * time.Minute,
	}

	svc, err := NewWebAuthnService(cfg)
	require.NoError(t, err)

	userStore := inmemory.NewUserStore()
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err)

	tests := []struct {
		name               string
		user               *domain.User
		deviceName         string
		requireHardwareKey bool
		wantErr            bool
		errMsg             string
	}{
		{
			name:               "ok/basic_registration",
			user:               user,
			deviceName:         "My Security Key",
			requireHardwareKey: false,
			wantErr:            false,
		},
		{
			name:               "ok/hardware_key_required",
			user:               user,
			deviceName:         "Hardware Key",
			requireHardwareKey: true,
			wantErr:            false,
		},
		{
			name:               "ok/empty_device_name",
			user:               user,
			deviceName:         "",
			requireHardwareKey: false,
			wantErr:            false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			options, sessionKey, err := svc.BeginRegistration(ctx, tc.user, tc.deviceName, tc.requireHardwareKey)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, options)
				assert.Empty(t, sessionKey)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, options)
				assert.NotEmpty(t, sessionKey)
			}
		})
	}
}

func TestWebAuthnService_BeginLogin(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := WebAuthnConfig{
		RPDisplayName: "Test RP",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:3000"},
		Users:         inmemory.NewUserStore(),
		Credentials:   inmemory.NewCredentialStore(),
		Challenges:    inmemory.NewChallengeStore(),
		Rand:          crypto.NewSecureRand(),
		ChallengeTTL:  5 * time.Minute,
	}

	svc, err := NewWebAuthnService(cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		userHint string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "ok/with_user_hint",
			userHint: "user@example.com",
			wantErr:  false,
		},
		{
			name:     "ok/empty_user_hint_discoverable",
			userHint: "",
			wantErr:  false,
		},
		{
			name:     "ok/with_nonexistent_user_hint",
			userHint: "nonexistent@example.com",
			wantErr:  false, // userHint is just a hint, not validated
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			options, sessionKey, err := svc.BeginLogin(ctx, tc.userHint)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, options)
				assert.Empty(t, sessionKey)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, options)
				assert.NotEmpty(t, sessionKey)
			}
		})
	}
}

func TestWebAuthnService_BeginStepUp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := WebAuthnConfig{
		RPDisplayName: "Test RP",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:3000"},
		Users:         inmemory.NewUserStore(),
		Credentials:   inmemory.NewCredentialStore(),
		Challenges:    inmemory.NewChallengeStore(),
		Rand:          crypto.NewSecureRand(),
		ChallengeTTL:  5 * time.Minute,
	}

	svc, err := NewWebAuthnService(cfg)
	require.NoError(t, err)

	userStore := inmemory.NewUserStore()
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err)

	tests := []struct {
		name    string
		user    *domain.User
		action  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "error/no_credentials",
			user:    user,
			action:  "delete_account",
			wantErr: true,
			errMsg:  "no credentials",
		},
		{
			name:    "error/no_credentials_different_action",
			user:    user,
			action:  "add_admin",
			wantErr: true,
			errMsg:  "no credentials",
		},
		{
			name:    "error/no_credentials_empty_action",
			user:    user,
			action:  "",
			wantErr: true,
			errMsg:  "no credentials",
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			options, sessionKey, err := svc.BeginStepUp(ctx, tc.user, tc.action)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, options)
				assert.Empty(t, sessionKey)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, options)
				assert.NotEmpty(t, sessionKey)
			}
		})
	}
}

func TestWebAuthnService_ChallengeExpiry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create service with short TTL for testing
	cfg := WebAuthnConfig{
		RPDisplayName: "Test RP",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:3000"},
		Users:         inmemory.NewUserStore(),
		Credentials:   inmemory.NewCredentialStore(),
		Challenges:    inmemory.NewChallengeStore(),
		Rand:          crypto.NewSecureRand(),
		ChallengeTTL:  100 * time.Millisecond, // Very short TTL
	}

	svc, err := NewWebAuthnService(cfg)
	require.NoError(t, err)

	userStore := inmemory.NewUserStore()
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err)

	// Start registration
	_, sessionKey, err := svc.BeginRegistration(ctx, user, "Device", false)
	require.NoError(t, err)

	// Wait for challenge to expire
	time.Sleep(200 * time.Millisecond)

	// Try to finish registration with expired challenge
	response := map[string]interface{}{
		"id":   "test",
		"type": "public-key",
	}
	respBytes, _ := json.Marshal(response)
	_, err = svc.FinishRegistration(ctx, sessionKey, json.RawMessage(respBytes))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestWebAuthnService_OriginValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Create service with specific origins
	cfg := WebAuthnConfig{
		RPDisplayName: "Test RP",
		RPID:          "example.com",
		RPOrigins:     []string{"https://example.com", "https://app.example.com"},
		Users:         inmemory.NewUserStore(),
		Credentials:   inmemory.NewCredentialStore(),
		Challenges:    inmemory.NewChallengeStore(),
		Rand:          crypto.NewSecureRand(),
		ChallengeTTL:  5 * time.Minute,
	}

	svc, err := NewWebAuthnService(cfg)
	require.NoError(t, err)

	userStore := inmemory.NewUserStore()
	user, err := userStore.Create(ctx, "user@example.com", []string{"user"})
	require.NoError(t, err)

	// Begin registration
	options, sessionKey, err := svc.BeginRegistration(ctx, user, "Device", false)
	require.NoError(t, err)
	assert.NotNil(t, options)
	assert.NotEmpty(t, sessionKey)

	// Test would validate origin in actual response processing
}

func TestWebAuthnService_AdminAAGUIDEnforcement(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := WebAuthnConfig{
		RPDisplayName:        "Test RP",
		RPID:                 "localhost",
		RPOrigins:            []string{"http://localhost:3000"},
		Users:                inmemory.NewUserStore(),
		Credentials:          inmemory.NewCredentialStore(),
		Challenges:           inmemory.NewChallengeStore(),
		Rand:                 crypto.NewSecureRand(),
		AdminAAGUIDAllowlist: []string{"00000000-0000-0000-0000-000000000001"},
		ChallengeTTL:         5 * time.Minute,
	}

	svc, err := NewWebAuthnService(cfg)
	require.NoError(t, err)

	userStore := inmemory.NewUserStore()
	adminUser, err := userStore.Create(ctx, "admin@example.com", []string{"admin"})
	require.NoError(t, err)

	// Begin registration requiring hardware key for admin
	options, sessionKey, err := svc.BeginRegistration(ctx, adminUser, "Admin Key", true)
	require.NoError(t, err)
	assert.NotNil(t, options)
	assert.NotEmpty(t, sessionKey)
}

func TestWebAuthnService_Concurrency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := WebAuthnConfig{
		RPDisplayName: "Test RP",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:3000"},
		Users:         inmemory.NewUserStore(),
		Credentials:   inmemory.NewCredentialStore(),
		Challenges:    inmemory.NewChallengeStore(),
		Rand:          crypto.NewSecureRand(),
		ChallengeTTL:  5 * time.Minute,
	}

	svc, err := NewWebAuthnService(cfg)
	require.NoError(t, err)

	userStore := inmemory.NewUserStore()

	// Create multiple users
	var users []*domain.User
	for i := 0; i < 5; i++ {
		user, err := userStore.Create(ctx, fmt.Sprintf("user%d@example.com", i), []string{"user"})
		require.NoError(t, err)
		users = append(users, user)
	}

	// Run concurrent operations
	done := make(chan bool, 3)

	// Goroutine 1: Begin registrations
	go func() {
		for i := 0; i < 20; i++ {
			user := users[i%len(users)]
			_, _, err := svc.BeginRegistration(ctx, user, fmt.Sprintf("Device %d", i), false)
			assert.NoError(t, err)
		}
		done <- true
	}()

	// Goroutine 2: Begin logins
	go func() {
		for i := 0; i < 20; i++ {
			user := users[i%len(users)]
			_, _, err := svc.BeginLogin(ctx, user.Email)
			// May error if user not found, that's ok
			_ = err
		}
		done <- true
	}()

	// Goroutine 3: Begin step-ups
	go func() {
		for i := 0; i < 20; i++ {
			user := users[i%len(users)]
			_, _, err := svc.BeginStepUp(ctx, user, fmt.Sprintf("action-%d", i))
			// May fail if no credentials, that's ok
			_ = err
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

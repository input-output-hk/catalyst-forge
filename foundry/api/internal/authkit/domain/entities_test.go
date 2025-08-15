package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name string
		user User
	}{
		{
			name: "complete_user",
			user: User{
				ID:             uuid.New(),
				Email:          "user@example.com",
				Roles:          []string{"user", "admin"},
				Permissions:    []string{"read", "write", "delete"},
				SessionVersion: 42,
				SuspendedAt:    timePtr(time.Now().UTC()),
				CreatedAt:      time.Now().UTC(),
				UpdatedAt:      time.Now().UTC(),
			},
		},
		{
			name: "minimal_user",
			user: User{
				ID:             uuid.New(),
				Email:          "minimal@example.com",
				Roles:          []string{},
				Permissions:    []string{},
				SessionVersion: 1,
				SuspendedAt:    nil,
				CreatedAt:      time.Now().UTC(),
				UpdatedAt:      time.Now().UTC(),
			},
		},
		{
			name: "suspended_user",
			user: User{
				ID:             uuid.New(),
				Email:          "suspended@example.com",
				Roles:          []string{"user"},
				Permissions:    []string{"read"},
				SessionVersion: 5,
				SuspendedAt:    timePtr(time.Now().UTC().Add(-24 * time.Hour)),
				CreatedAt:      time.Now().UTC().Add(-48 * time.Hour),
				UpdatedAt:      time.Now().UTC().Add(-24 * time.Hour),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.user)
			require.NoError(t, err, "marshaling should succeed")
			assert.NotEmpty(t, data, "marshaled data should not be empty")

			// Test unmarshaling
			var unmarshaled User
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err, "unmarshaling should succeed")

			// Verify fields match
			assert.Equal(t, tt.user.ID, unmarshaled.ID)
			assert.Equal(t, tt.user.Email, unmarshaled.Email)
			assert.Equal(t, tt.user.Roles, unmarshaled.Roles)
			assert.Equal(t, tt.user.Permissions, unmarshaled.Permissions)
			assert.Equal(t, tt.user.SessionVersion, unmarshaled.SessionVersion)
			
			// Handle time comparison with tolerance for JSON marshaling precision
			assert.WithinDuration(t, tt.user.CreatedAt, unmarshaled.CreatedAt, time.Second)
			assert.WithinDuration(t, tt.user.UpdatedAt, unmarshaled.UpdatedAt, time.Second)
			
			if tt.user.SuspendedAt != nil {
				require.NotNil(t, unmarshaled.SuspendedAt)
				assert.WithinDuration(t, *tt.user.SuspendedAt, *unmarshaled.SuspendedAt, time.Second)
			} else {
				assert.Nil(t, unmarshaled.SuspendedAt)
			}
		})
	}
}

func TestUser_ZeroValues(t *testing.T) {
	var user User

	// Test zero values
	assert.Equal(t, uuid.Nil, user.ID)
	assert.Empty(t, user.Email)
	assert.Nil(t, user.Roles)
	assert.Nil(t, user.Permissions)
	assert.Zero(t, user.SessionVersion)
	assert.Nil(t, user.SuspendedAt)
	assert.True(t, user.CreatedAt.IsZero())
	assert.True(t, user.UpdatedAt.IsZero())
}

func TestUser_FieldConstraints(t *testing.T) {
	user := User{
		ID:             uuid.New(),
		Email:          "test@example.com",
		Roles:          []string{"user", "admin", "moderator"},
		Permissions:    []string{"read", "write", "delete", "admin"},
		SessionVersion: 999999,
		SuspendedAt:    timePtr(time.Now().UTC()),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Test that all fields can be set and retrieved
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.Contains(t, user.Email, "@")
	assert.Len(t, user.Roles, 3)
	assert.Len(t, user.Permissions, 4)
	assert.Greater(t, user.SessionVersion, int64(0))
	assert.NotNil(t, user.SuspendedAt)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestCredential_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name string
		cred Credential
	}{
		{
			name: "complete_credential",
			cred: Credential{
				ID:         []byte("credential-id-12345678"),
				UserID:     uuid.New(),
				PublicKey:  []byte("public-key-data-here"),
				AAGUID:     "12345678-1234-1234-1234-123456789012",
				DeviceName: "MacBook Pro Touch ID",
				RK:         true,
				Transports: []string{"internal", "usb"},
				SignCount:  42,
				CreatedAt:  time.Now().UTC(),
				LastUsedAt: time.Now().UTC().Add(-time.Hour),
				Revoked:    false,
			},
		},
		{
			name: "minimal_credential",
			cred: Credential{
				ID:         []byte("minimal-id"),
				UserID:     uuid.New(),
				PublicKey:  []byte("minimal-key"),
				AAGUID:     "",
				DeviceName: "Security Key",
				RK:         false,
				Transports: []string{},
				SignCount:  0,
				CreatedAt:  time.Now().UTC(),
				LastUsedAt: time.Now().UTC(),
				Revoked:    false,
			},
		},
		{
			name: "revoked_credential",
			cred: Credential{
				ID:         []byte("revoked-credential-id"),
				UserID:     uuid.New(),
				PublicKey:  []byte("revoked-public-key"),
				AAGUID:     "revoked-aaguid",
				DeviceName: "Revoked Device",
				RK:         true,
				Transports: []string{"nfc", "ble"},
				SignCount:  100,
				CreatedAt:  time.Now().UTC().Add(-48 * time.Hour),
				LastUsedAt: time.Now().UTC().Add(-24 * time.Hour),
				Revoked:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.cred)
			require.NoError(t, err, "marshaling should succeed")
			assert.NotEmpty(t, data, "marshaled data should not be empty")

			// Test unmarshaling
			var unmarshaled Credential
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err, "unmarshaling should succeed")

			// Verify fields match
			assert.Equal(t, tt.cred.ID, unmarshaled.ID)
			assert.Equal(t, tt.cred.UserID, unmarshaled.UserID)
			assert.Equal(t, tt.cred.PublicKey, unmarshaled.PublicKey)
			assert.Equal(t, tt.cred.AAGUID, unmarshaled.AAGUID)
			assert.Equal(t, tt.cred.DeviceName, unmarshaled.DeviceName)
			assert.Equal(t, tt.cred.RK, unmarshaled.RK)
			assert.Equal(t, tt.cred.Transports, unmarshaled.Transports)
			assert.Equal(t, tt.cred.SignCount, unmarshaled.SignCount)
			assert.Equal(t, tt.cred.Revoked, unmarshaled.Revoked)
			
			// Time fields with tolerance
			assert.WithinDuration(t, tt.cred.CreatedAt, unmarshaled.CreatedAt, time.Second)
			assert.WithinDuration(t, tt.cred.LastUsedAt, unmarshaled.LastUsedAt, time.Second)
		})
	}
}

func TestCredential_ZeroValues(t *testing.T) {
	var cred Credential

	// Test zero values
	assert.Nil(t, cred.ID)
	assert.Equal(t, uuid.Nil, cred.UserID)
	assert.Nil(t, cred.PublicKey)
	assert.Empty(t, cred.AAGUID)
	assert.Empty(t, cred.DeviceName)
	assert.False(t, cred.RK)
	assert.Nil(t, cred.Transports)
	assert.Zero(t, cred.SignCount)
	assert.True(t, cred.CreatedAt.IsZero())
	assert.True(t, cred.LastUsedAt.IsZero())
	assert.False(t, cred.Revoked)
}

func TestCredential_FieldConstraints(t *testing.T) {
	// Test edge cases and constraints
	tests := []struct {
		name string
		cred Credential
		desc string
	}{
		{
			name: "empty_id",
			cred: Credential{
				ID:        []byte{},
				UserID:    uuid.New(),
				PublicKey: []byte("key"),
			},
			desc: "Empty credential ID should be allowed",
		},
		{
			name: "large_sign_count",
			cred: Credential{
				ID:        []byte("test"),
				UserID:    uuid.New(),
				PublicKey: []byte("key"),
				SignCount: 4294967295, // Max uint32
			},
			desc: "Maximum sign count should be allowed",
		},
		{
			name: "multiple_transports",
			cred: Credential{
				ID:         []byte("test"),
				UserID:     uuid.New(),
				PublicKey:  []byte("key"),
				Transports: []string{"usb", "nfc", "ble", "internal"},
			},
			desc: "Multiple transport methods should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the credential can be created and fields accessed
			assert.NotNil(t, tt.cred.ID, tt.desc)
			assert.NotEqual(t, uuid.Nil, tt.cred.UserID, tt.desc)
		})
	}
}

func TestInvite_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name   string
		invite Invite
	}{
		{
			name: "active_invite",
			invite: Invite{
				ID:         uuid.New(),
				Email:      "newuser@company.com",
				Roles:      []string{"user", "viewer"},
				TokenHash:  []byte("hashed-token-data-here"),
				ExpiresAt:  time.Now().UTC().Add(24 * time.Hour),
				Attempts:   0,
				RedeemedAt: nil,
				CreatedBy:  uuid.New(),
			},
		},
		{
			name: "expired_invite",
			invite: Invite{
				ID:         uuid.New(),
				Email:      "expired@company.com",
				Roles:      []string{"user"},
				TokenHash:  []byte("expired-token-hash"),
				ExpiresAt:  time.Now().UTC().Add(-24 * time.Hour),
				Attempts:   3,
				RedeemedAt: nil,
				CreatedBy:  uuid.New(),
			},
		},
		{
			name: "redeemed_invite",
			invite: Invite{
				ID:         uuid.New(),
				Email:      "redeemed@company.com",
				Roles:      []string{"admin", "user"},
				TokenHash:  []byte("redeemed-token-hash"),
				ExpiresAt:  time.Now().UTC().Add(12 * time.Hour),
				Attempts:   1,
				RedeemedAt: timePtr(time.Now().UTC().Add(-time.Hour)),
				CreatedBy:  uuid.New(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.invite)
			require.NoError(t, err, "marshaling should succeed")
			assert.NotEmpty(t, data, "marshaled data should not be empty")

			// Test unmarshaling
			var unmarshaled Invite
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err, "unmarshaling should succeed")

			// Verify fields match
			assert.Equal(t, tt.invite.ID, unmarshaled.ID)
			assert.Equal(t, tt.invite.Email, unmarshaled.Email)
			assert.Equal(t, tt.invite.Roles, unmarshaled.Roles)
			assert.Equal(t, tt.invite.TokenHash, unmarshaled.TokenHash)
			assert.Equal(t, tt.invite.Attempts, unmarshaled.Attempts)
			assert.Equal(t, tt.invite.CreatedBy, unmarshaled.CreatedBy)
			
			// Time fields with tolerance
			assert.WithinDuration(t, tt.invite.ExpiresAt, unmarshaled.ExpiresAt, time.Second)
			
			if tt.invite.RedeemedAt != nil {
				require.NotNil(t, unmarshaled.RedeemedAt)
				assert.WithinDuration(t, *tt.invite.RedeemedAt, *unmarshaled.RedeemedAt, time.Second)
			} else {
				assert.Nil(t, unmarshaled.RedeemedAt)
			}
		})
	}
}

func TestInvite_ZeroValues(t *testing.T) {
	var invite Invite

	// Test zero values
	assert.Equal(t, uuid.Nil, invite.ID)
	assert.Empty(t, invite.Email)
	assert.Nil(t, invite.Roles)
	assert.Nil(t, invite.TokenHash)
	assert.True(t, invite.ExpiresAt.IsZero())
	assert.Zero(t, invite.Attempts)
	assert.Nil(t, invite.RedeemedAt)
	assert.Equal(t, uuid.Nil, invite.CreatedBy)
}

func TestInvite_FieldConstraints(t *testing.T) {
	// Test invite with various constraints
	invite := Invite{
		ID:         uuid.New(),
		Email:      "test@example.com",
		Roles:      []string{"user", "admin", "moderator", "viewer"},
		TokenHash:  make([]byte, 32), // 256-bit hash
		ExpiresAt:  time.Now().UTC().Add(7 * 24 * time.Hour), // Week expiry
		Attempts:   5, // Failed attempts
		RedeemedAt: nil,
		CreatedBy:  uuid.New(),
	}

	// Verify constraints
	assert.NotEqual(t, uuid.Nil, invite.ID)
	assert.Contains(t, invite.Email, "@")
	assert.Greater(t, len(invite.Roles), 0)
	assert.Len(t, invite.TokenHash, 32) // HMAC-SHA256 size
	assert.True(t, invite.ExpiresAt.After(time.Now()))
	assert.GreaterOrEqual(t, invite.Attempts, 0)
	assert.NotEqual(t, uuid.Nil, invite.CreatedBy)
}

func TestRecoveryCode_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name string
		code RecoveryCode
	}{
		{
			name: "unused_code",
			code: RecoveryCode{
				UserID:    uuid.New(),
				Hash:      []byte("sha256-hash-of-recovery-code"),
				UsedAt:    nil,
				CreatedAt: time.Now().UTC(),
			},
		},
		{
			name: "used_code",
			code: RecoveryCode{
				UserID:    uuid.New(),
				Hash:      []byte("sha256-hash-of-used-code"),
				UsedAt:    timePtr(time.Now().UTC().Add(-time.Hour)),
				CreatedAt: time.Now().UTC().Add(-24 * time.Hour),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.code)
			require.NoError(t, err, "marshaling should succeed")
			assert.NotEmpty(t, data, "marshaled data should not be empty")

			// Test unmarshaling
			var unmarshaled RecoveryCode
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err, "unmarshaling should succeed")

			// Verify fields match
			assert.Equal(t, tt.code.UserID, unmarshaled.UserID)
			assert.Equal(t, tt.code.Hash, unmarshaled.Hash)
			assert.WithinDuration(t, tt.code.CreatedAt, unmarshaled.CreatedAt, time.Second)
			
			if tt.code.UsedAt != nil {
				require.NotNil(t, unmarshaled.UsedAt)
				assert.WithinDuration(t, *tt.code.UsedAt, *unmarshaled.UsedAt, time.Second)
			} else {
				assert.Nil(t, unmarshaled.UsedAt)
			}
		})
	}
}

func TestRecoveryCode_ZeroValues(t *testing.T) {
	var code RecoveryCode

	// Test zero values
	assert.Equal(t, uuid.Nil, code.UserID)
	assert.Nil(t, code.Hash)
	assert.Nil(t, code.UsedAt)
	assert.True(t, code.CreatedAt.IsZero())
}

func TestRefreshToken_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name  string
		token RefreshToken
	}{
		{
			name: "active_token",
			token: RefreshToken{
				ID:             uuid.New(),
				FamilyID:       uuid.New(),
				UserID:         uuid.New(),
				Hash:           []byte("sha256-hash-of-token"),
				SessionVersion: 42,
				CreatedAt:      time.Now().UTC(),
				ExpiresAt:      time.Now().UTC().Add(30 * 24 * time.Hour),
				RotatedAt:      nil,
				RevokedAt:      nil,
				Reason:         nil,
			},
		},
		{
			name: "rotated_token",
			token: RefreshToken{
				ID:             uuid.New(),
				FamilyID:       uuid.New(),
				UserID:         uuid.New(),
				Hash:           []byte("sha256-hash-of-rotated-token"),
				SessionVersion: 43,
				CreatedAt:      time.Now().UTC().Add(-time.Hour),
				ExpiresAt:      time.Now().UTC().Add(29 * 24 * time.Hour),
				RotatedAt:      timePtr(time.Now().UTC().Add(-30 * time.Minute)),
				RevokedAt:      nil,
				Reason:         nil,
			},
		},
		{
			name: "revoked_token",
			token: RefreshToken{
				ID:             uuid.New(),
				FamilyID:       uuid.New(),
				UserID:         uuid.New(),
				Hash:           []byte("sha256-hash-of-revoked-token"),
				SessionVersion: 40,
				CreatedAt:      time.Now().UTC().Add(-2 * time.Hour),
				ExpiresAt:      time.Now().UTC().Add(28 * 24 * time.Hour),
				RotatedAt:      nil,
				RevokedAt:      timePtr(time.Now().UTC().Add(-time.Hour)),
				Reason:         stringPtr("security_violation"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.token)
			require.NoError(t, err, "marshaling should succeed")
			assert.NotEmpty(t, data, "marshaled data should not be empty")

			// Test unmarshaling
			var unmarshaled RefreshToken
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err, "unmarshaling should succeed")

			// Verify fields match
			assert.Equal(t, tt.token.ID, unmarshaled.ID)
			assert.Equal(t, tt.token.FamilyID, unmarshaled.FamilyID)
			assert.Equal(t, tt.token.UserID, unmarshaled.UserID)
			assert.Equal(t, tt.token.Hash, unmarshaled.Hash)
			assert.Equal(t, tt.token.SessionVersion, unmarshaled.SessionVersion)
			
			// Time fields with tolerance
			assert.WithinDuration(t, tt.token.CreatedAt, unmarshaled.CreatedAt, time.Second)
			assert.WithinDuration(t, tt.token.ExpiresAt, unmarshaled.ExpiresAt, time.Second)
			
			if tt.token.RotatedAt != nil {
				require.NotNil(t, unmarshaled.RotatedAt)
				assert.WithinDuration(t, *tt.token.RotatedAt, *unmarshaled.RotatedAt, time.Second)
			} else {
				assert.Nil(t, unmarshaled.RotatedAt)
			}
			
			if tt.token.RevokedAt != nil {
				require.NotNil(t, unmarshaled.RevokedAt)
				assert.WithinDuration(t, *tt.token.RevokedAt, *unmarshaled.RevokedAt, time.Second)
			} else {
				assert.Nil(t, unmarshaled.RevokedAt)
			}
			
			if tt.token.Reason != nil {
				require.NotNil(t, unmarshaled.Reason)
				assert.Equal(t, *tt.token.Reason, *unmarshaled.Reason)
			} else {
				assert.Nil(t, unmarshaled.Reason)
			}
		})
	}
}

func TestRefreshToken_ZeroValues(t *testing.T) {
	var token RefreshToken

	// Test zero values
	assert.Equal(t, uuid.Nil, token.ID)
	assert.Equal(t, uuid.Nil, token.FamilyID)
	assert.Equal(t, uuid.Nil, token.UserID)
	assert.Nil(t, token.Hash)
	assert.Zero(t, token.SessionVersion)
	assert.True(t, token.CreatedAt.IsZero())
	assert.True(t, token.ExpiresAt.IsZero())
	assert.Nil(t, token.RotatedAt)
	assert.Nil(t, token.RevokedAt)
	assert.Nil(t, token.Reason)
}

func TestRefreshToken_FieldConstraints(t *testing.T) {
	// Test token lifecycle states
	tests := []struct {
		name        string
		token       RefreshToken
		description string
	}{
		{
			name: "fresh_token",
			token: RefreshToken{
				ID:             uuid.New(),
				FamilyID:       uuid.New(),
				UserID:         uuid.New(),
				Hash:           make([]byte, 32), // SHA256 size
				SessionVersion: 1,
				CreatedAt:      time.Now().UTC(),
				ExpiresAt:      time.Now().UTC().Add(30 * 24 * time.Hour),
				RotatedAt:      nil,
				RevokedAt:      nil,
				Reason:         nil,
			},
			description: "Fresh token should have no rotation or revocation",
		},
		{
			name: "family_rotated_token",
			token: RefreshToken{
				ID:             uuid.New(),
				FamilyID:       uuid.New(),
				UserID:         uuid.New(),
				Hash:           make([]byte, 32),
				SessionVersion: 5,
				CreatedAt:      time.Now().UTC().Add(-time.Hour),
				ExpiresAt:      time.Now().UTC().Add(29 * 24 * time.Hour),
				RotatedAt:      timePtr(time.Now().UTC()),
				RevokedAt:      nil,
				Reason:         nil,
			},
			description: "Rotated token should have RotatedAt set but no RevokedAt",
		},
		{
			name: "revoked_token_with_reason",
			token: RefreshToken{
				ID:             uuid.New(),
				FamilyID:       uuid.New(),
				UserID:         uuid.New(),
				Hash:           make([]byte, 32),
				SessionVersion: 10,
				CreatedAt:      time.Now().UTC().Add(-2 * time.Hour),
				ExpiresAt:      time.Now().UTC().Add(28 * 24 * time.Hour),
				RotatedAt:      nil,
				RevokedAt:      timePtr(time.Now().UTC().Add(-time.Hour)),
				Reason:         stringPtr("replay_attack_detected"),
			},
			description: "Revoked token should have both RevokedAt and Reason set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify basic constraints
			assert.NotEqual(t, uuid.Nil, tt.token.ID, tt.description)
			assert.NotEqual(t, uuid.Nil, tt.token.FamilyID, tt.description)
			assert.NotEqual(t, uuid.Nil, tt.token.UserID, tt.description)
			assert.NotEmpty(t, tt.token.Hash, tt.description)
			assert.Greater(t, tt.token.SessionVersion, int64(0), tt.description)
			assert.True(t, tt.token.ExpiresAt.After(tt.token.CreatedAt), tt.description)
			
			// Verify lifecycle constraints
			if tt.token.RotatedAt != nil {
				assert.True(t, tt.token.RotatedAt.After(tt.token.CreatedAt), tt.description)
			}
			
			if tt.token.RevokedAt != nil {
				assert.True(t, tt.token.RevokedAt.After(tt.token.CreatedAt), tt.description)
				// Revoked tokens should have a reason
				if tt.name == "revoked_token_with_reason" {
					assert.NotNil(t, tt.token.Reason, tt.description)
					assert.NotEmpty(t, *tt.token.Reason, tt.description)
				}
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Test domain constants
	assert.Equal(t, "__Host-refresh-token", RefreshCookieName)
	assert.Contains(t, RefreshCookieName, "__Host-")
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}
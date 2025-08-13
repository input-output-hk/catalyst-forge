package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/store"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/testing/inmemory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStepUpService(t *testing.T) (StepUpService, store.KV) {
	t.Helper()
	
	kv := inmemory.NewKV()
	defaultTTL := 5 * time.Minute
	
	svc := NewStepUpService(kv, defaultTTL)
	return svc, kv
}

func TestNewStepUpService(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name       string
		kv         store.KV
		defaultTTL time.Duration
	}{
		{
			name:       "ok/with_default_ttl",
			kv:         inmemory.NewKV(),
			defaultTTL: 5 * time.Minute,
		},
		{
			name:       "ok/zero_ttl_uses_default",
			kv:         inmemory.NewKV(),
			defaultTTL: 0,
		},
		{
			name:       "ok/negative_ttl_uses_default",
			kv:         inmemory.NewKV(),
			defaultTTL: -1 * time.Minute,
		},
		{
			name:       "ok/nil_kv_store",
			kv:         nil,
			defaultTTL: 5 * time.Minute,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			assert.NotPanics(t, func() {
				svc := NewStepUpService(tc.kv, tc.defaultTTL)
				assert.NotNil(t, svc)
			})
		})
	}
}

func TestStepUpService_GrantStepUp(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		setup   func(t *testing.T) (StepUpService, uuid.UUID)
		action  string
		ttl     time.Duration
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/basic_grant",
			setup: func(t *testing.T) (StepUpService, uuid.UUID) {
				svc, _ := setupStepUpService(t)
				return svc, uuid.New()
			},
			action:  "delete_account",
			ttl:     5 * time.Minute,
			wantErr: false,
		},
		{
			name: "ok/zero_ttl_uses_default",
			setup: func(t *testing.T) (StepUpService, uuid.UUID) {
				svc, _ := setupStepUpService(t)
				return svc, uuid.New()
			},
			action:  "admin_action",
			ttl:     0,
			wantErr: false,
		},
		{
			name: "ok/ttl_capped_to_max",
			setup: func(t *testing.T) (StepUpService, uuid.UUID) {
				svc, _ := setupStepUpService(t)
				return svc, uuid.New()
			},
			action:  "long_action",
			ttl:     2 * time.Hour, // Will be capped to 30 minutes
			wantErr: false,
		},
		{
			name: "ok/empty_action",
			setup: func(t *testing.T) (StepUpService, uuid.UUID) {
				svc, _ := setupStepUpService(t)
				return svc, uuid.New()
			},
			action:  "",
			ttl:     5 * time.Minute,
			wantErr: false,
		},
		{
			name: "error/nil_kv_store",
			setup: func(t *testing.T) (StepUpService, uuid.UUID) {
				svc := NewStepUpService(nil, 5*time.Minute)
				return svc, uuid.New()
			},
			action:  "action",
			ttl:     5 * time.Minute,
			wantErr: true,
			errMsg:  "requires KV store",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			svc, userID := tc.setup(t)
			
			err := svc.GrantStepUp(ctx, userID, tc.action, tc.ttl)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify grant was created
				err = svc.ValidateStepUp(ctx, userID, tc.action)
				assert.NoError(t, err, "grant should be valid immediately after creation")
			}
		})
	}
}

func TestStepUpService_ValidateStepUp(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		setup   func(t *testing.T) (StepUpService, uuid.UUID, string)
		wantErr bool
		errType error
	}{
		{
			name: "ok/valid_grant",
			setup: func(t *testing.T) (StepUpService, uuid.UUID, string) {
				svc, _ := setupStepUpService(t)
				userID := uuid.New()
				action := "sensitive_action"
				
				// Grant step-up
				err := svc.GrantStepUp(ctx, userID, action, 5*time.Minute)
				require.NoError(t, err)
				
				return svc, userID, action
			},
			wantErr: false,
		},
		{
			name: "error/no_grant",
			setup: func(t *testing.T) (StepUpService, uuid.UUID, string) {
				svc, _ := setupStepUpService(t)
				return svc, uuid.New(), "unknown_action"
			},
			wantErr: true,
			errType: ErrStepUpRequired,
		},
		{
			name: "error/expired_grant",
			setup: func(t *testing.T) (StepUpService, uuid.UUID, string) {
				svc, kv := setupStepUpService(t)
				userID := uuid.New()
				action := "expired_action"
				
				// Grant with very short TTL
				err := svc.GrantStepUp(ctx, userID, action, 50*time.Millisecond)
				require.NoError(t, err)
				
				// Wait for expiry
				time.Sleep(100 * time.Millisecond)
				
				// Force KV to expire entries
				if kvImpl, ok := kv.(*inmemory.KV); ok {
					kvImpl.Cleanup()
				}
				
				return svc, userID, action
			},
			wantErr: true,
			errType: ErrStepUpRequired,
		},
		{
			name: "error/nil_kv_store",
			setup: func(t *testing.T) (StepUpService, uuid.UUID, string) {
				svc := NewStepUpService(nil, 5*time.Minute)
				return svc, uuid.New(), "action"
			},
			wantErr: true,
			errType: ErrStepUpRequired,
		},
		{
			name: "ok/empty_action_default",
			setup: func(t *testing.T) (StepUpService, uuid.UUID, string) {
				svc, _ := setupStepUpService(t)
				userID := uuid.New()
				
				// Grant with empty action (becomes "default")
				err := svc.GrantStepUp(ctx, userID, "", 5*time.Minute)
				require.NoError(t, err)
				
				return svc, userID, ""
			},
			wantErr: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			svc, userID, action := tc.setup(t)
			
			err := svc.ValidateStepUp(ctx, userID, action)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errType != nil {
					assert.ErrorIs(t, err, tc.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStepUpService_RevokeStepUp(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		setup   func(t *testing.T) (StepUpService, uuid.UUID, string)
		wantErr bool
	}{
		{
			name: "ok/revoke_existing_grant",
			setup: func(t *testing.T) (StepUpService, uuid.UUID, string) {
				svc, _ := setupStepUpService(t)
				userID := uuid.New()
				action := "revokable_action"
				
				// Grant step-up
				err := svc.GrantStepUp(ctx, userID, action, 5*time.Minute)
				require.NoError(t, err)
				
				return svc, userID, action
			},
			wantErr: false,
		},
		{
			name: "ok/revoke_nonexistent_grant",
			setup: func(t *testing.T) (StepUpService, uuid.UUID, string) {
				svc, _ := setupStepUpService(t)
				return svc, uuid.New(), "nonexistent_action"
			},
			wantErr: false,
		},
		{
			name: "ok/nil_kv_store",
			setup: func(t *testing.T) (StepUpService, uuid.UUID, string) {
				svc := NewStepUpService(nil, 5*time.Minute)
				return svc, uuid.New(), "action"
			},
			wantErr: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			svc, userID, action := tc.setup(t)
			
			err := svc.RevokeStepUp(ctx, userID, action)
			
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				
				// Verify grant was revoked
				err = svc.ValidateStepUp(ctx, userID, action)
				assert.ErrorIs(t, err, ErrStepUpRequired, "grant should be invalid after revocation")
			}
		})
	}
}

func TestStepUpService_RevokeAllStepUps(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, _ := setupStepUpService(t)
	userID := uuid.New()
	
	// Grant multiple step-ups
	actions := []string{"action1", "action2", "sensitive", "admin", "delete"}
	for _, action := range actions {
		err := svc.GrantStepUp(ctx, userID, action, 5*time.Minute)
		require.NoError(t, err)
	}
	
	// Verify all grants are valid
	for _, action := range actions {
		err := svc.ValidateStepUp(ctx, userID, action)
		assert.NoError(t, err, "grant for %s should be valid", action)
	}
	
	// Revoke all grants
	err := svc.RevokeAllStepUps(ctx, userID)
	require.NoError(t, err)
	
	// Verify common actions are revoked
	commonActions := []string{"sensitive", "admin", "delete"}
	for _, action := range commonActions {
		err := svc.ValidateStepUp(ctx, userID, action)
		assert.ErrorIs(t, err, ErrStepUpRequired, "grant for %s should be revoked", action)
	}
}

func TestStepUpService_TTLBehavior(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("ttl_expiry", func(t *testing.T) {
		t.Parallel()
		
		svc, kv := setupStepUpService(t)
		userID := uuid.New()
		action := "short_lived"
		
		// Grant with very short TTL
		err := svc.GrantStepUp(ctx, userID, action, 100*time.Millisecond)
		require.NoError(t, err)
		
		// Should be valid immediately
		err = svc.ValidateStepUp(ctx, userID, action)
		assert.NoError(t, err)
		
		// Wait for expiry
		time.Sleep(150 * time.Millisecond)
		
		// Force cleanup of expired entries
		if kvImpl, ok := kv.(*inmemory.KV); ok {
			kvImpl.Cleanup()
		}
		
		// Should be expired
		err = svc.ValidateStepUp(ctx, userID, action)
		assert.ErrorIs(t, err, ErrStepUpRequired)
	})
	
	t.Run("max_ttl_cap", func(t *testing.T) {
		t.Parallel()
		
		svc, _ := setupStepUpService(t)
		userID := uuid.New()
		action := "long_lived"
		
		// Try to grant with very long TTL (should be capped)
		err := svc.GrantStepUp(ctx, userID, action, 24*time.Hour)
		require.NoError(t, err)
		
		// Should be valid
		err = svc.ValidateStepUp(ctx, userID, action)
		assert.NoError(t, err)
		// Note: We can't easily test the actual cap without mocking time
	})
}

func TestStepUpService_IsolationByUserAndAction(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, _ := setupStepUpService(t)
	
	user1 := uuid.New()
	user2 := uuid.New()
	action1 := "action1"
	action2 := "action2"
	
	// Grant step-up for user1/action1
	err := svc.GrantStepUp(ctx, user1, action1, 5*time.Minute)
	require.NoError(t, err)
	
	// user1/action1 should be valid
	err = svc.ValidateStepUp(ctx, user1, action1)
	assert.NoError(t, err)
	
	// user1/action2 should NOT be valid
	err = svc.ValidateStepUp(ctx, user1, action2)
	assert.ErrorIs(t, err, ErrStepUpRequired)
	
	// user2/action1 should NOT be valid
	err = svc.ValidateStepUp(ctx, user2, action1)
	assert.ErrorIs(t, err, ErrStepUpRequired)
	
	// Grant step-up for user2/action2
	err = svc.GrantStepUp(ctx, user2, action2, 5*time.Minute)
	require.NoError(t, err)
	
	// user2/action2 should be valid
	err = svc.ValidateStepUp(ctx, user2, action2)
	assert.NoError(t, err)
	
	// user1/action1 should still be valid
	err = svc.ValidateStepUp(ctx, user1, action1)
	assert.NoError(t, err)
	
	// Revoke user1/action1
	err = svc.RevokeStepUp(ctx, user1, action1)
	require.NoError(t, err)
	
	// user1/action1 should NOT be valid
	err = svc.ValidateStepUp(ctx, user1, action1)
	assert.ErrorIs(t, err, ErrStepUpRequired)
	
	// user2/action2 should still be valid
	err = svc.ValidateStepUp(ctx, user2, action2)
	assert.NoError(t, err)
}

func TestStepUpService_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, _ := setupStepUpService(t)
	
	// Create multiple users and actions
	const numUsers = 5
	const numActions = 3
	users := make([]uuid.UUID, numUsers)
	for i := 0; i < numUsers; i++ {
		users[i] = uuid.New()
	}
	actions := []string{"read", "write", "delete"}
	
	var wg sync.WaitGroup
	
	// Grant step-ups concurrently
	for i := 0; i < numUsers; i++ {
		for j := 0; j < numActions; j++ {
			wg.Add(1)
			go func(userID uuid.UUID, action string) {
				defer wg.Done()
				err := svc.GrantStepUp(ctx, userID, action, 5*time.Minute)
				assert.NoError(t, err)
			}(users[i], actions[j])
		}
	}
	wg.Wait()
	
	// Validate step-ups concurrently
	for i := 0; i < numUsers; i++ {
		for j := 0; j < numActions; j++ {
			wg.Add(1)
			go func(userID uuid.UUID, action string) {
				defer wg.Done()
				err := svc.ValidateStepUp(ctx, userID, action)
				assert.NoError(t, err)
			}(users[i], actions[j])
		}
	}
	wg.Wait()
	
	// Revoke some step-ups concurrently
	for i := 0; i < numUsers; i += 2 {
		for j := 0; j < numActions; j += 2 {
			wg.Add(1)
			go func(userID uuid.UUID, action string) {
				defer wg.Done()
				err := svc.RevokeStepUp(ctx, userID, action)
				assert.NoError(t, err)
			}(users[i], actions[j])
		}
	}
	wg.Wait()
	
	// Verify revocations
	for i := 0; i < numUsers; i++ {
		for j := 0; j < numActions; j++ {
			err := svc.ValidateStepUp(ctx, users[i], actions[j])
			if i%2 == 0 && j%2 == 0 {
				// Should be revoked
				assert.ErrorIs(t, err, ErrStepUpRequired)
			} else {
				// Should still be valid
				assert.NoError(t, err)
			}
		}
	}
}

func TestStepUpService_OverwriteBehavior(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, _ := setupStepUpService(t)
	userID := uuid.New()
	action := "overwrite_test"
	
	// Grant with short TTL
	err := svc.GrantStepUp(ctx, userID, action, 1*time.Second)
	require.NoError(t, err)
	
	// Immediately grant again with longer TTL (should overwrite)
	err = svc.GrantStepUp(ctx, userID, action, 5*time.Minute)
	require.NoError(t, err)
	
	// Wait for original TTL to pass
	time.Sleep(1500 * time.Millisecond)
	
	// Should still be valid (using new TTL)
	err = svc.ValidateStepUp(ctx, userID, action)
	assert.NoError(t, err, "grant should use the new TTL")
}

func TestStepUpService_InvalidTimestamp(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	kv := inmemory.NewKV()
	svc := NewStepUpService(kv, 5*time.Minute)
	userID := uuid.New()
	action := "corrupted"
	
	// Manually store invalid timestamp
	key := "stepup:" + userID.String() + ":" + action
	err := kv.Set(ctx, key, []byte("invalid-timestamp"), 5*time.Minute)
	require.NoError(t, err)
	
	// Validation should treat as expired
	err = svc.ValidateStepUp(ctx, userID, action)
	assert.ErrorIs(t, err, ErrStepUpExpired)
}

func TestStepUpService_StaleEntry(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	kv := inmemory.NewKV()
	svc := NewStepUpService(kv, 5*time.Minute)
	userID := uuid.New()
	action := "stale"
	
	// Manually store old timestamp (beyond max TTL)
	oldTime := time.Now().UTC().Add(-1 * time.Hour)
	key := "stepup:" + userID.String() + ":" + action
	err := kv.Set(ctx, key, []byte(oldTime.Format(time.RFC3339)), 2*time.Hour)
	require.NoError(t, err)
	
	// Validation should treat as expired and clean up
	err = svc.ValidateStepUp(ctx, userID, action)
	assert.ErrorIs(t, err, ErrStepUpExpired)
	
	// Entry should be cleaned up
	_, err = kv.Get(ctx, key)
	assert.ErrorIs(t, err, store.ErrNotFound)
}
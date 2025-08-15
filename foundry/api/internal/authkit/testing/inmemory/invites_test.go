package inmemory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInviteStore(t *testing.T) {
	t.Parallel()
	
	store := NewInviteStore()
	assert.NotNil(t, store)
}

func TestInviteStore_Create(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	inviteID := uuid.New()
	createdBy := uuid.New()
	future := time.Now().Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		invite  *domain.Invite
		setup   func(*InviteStore)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/basic_invite",
			invite: &domain.Invite{
				ID:        inviteID,
				Email:     "user@example.com",
				Roles:     []string{"user"},
				TokenHash: []byte("token-hash-123"),
				ExpiresAt: future,
				Attempts:  0,
				CreatedBy: createdBy,
			},
			wantErr: false,
		},
		{
			name: "ok/multiple_roles",
			invite: &domain.Invite{
				ID:        uuid.New(),
				Email:     "admin@example.com",
				Roles:     []string{"user", "admin", "moderator"},
				TokenHash: []byte("admin-token-hash"),
				ExpiresAt: future,
				Attempts:  0,
				CreatedBy: createdBy,
			},
			wantErr: false,
		},
		{
			name: "ok/no_roles",
			invite: &domain.Invite{
				ID:        uuid.New(),
				Email:     "guest@example.com",
				Roles:     nil,
				TokenHash: []byte("guest-token-hash"),
				ExpiresAt: future,
				Attempts:  0,
				CreatedBy: createdBy,
			},
			wantErr: false,
		},
		{
			name: "error/duplicate_invite",
			invite: &domain.Invite{
				ID:        inviteID,
				Email:     "duplicate@example.com",
				Roles:     []string{"user"},
				TokenHash: []byte("duplicate-hash"),
				ExpiresAt: future,
				CreatedBy: createdBy,
			},
			setup: func(store *InviteStore) {
				err := store.Create(ctx, &domain.Invite{
					ID:        inviteID,
					Email:     "original@example.com",
					Roles:     []string{"admin"},
					TokenHash: []byte("original-hash"),
					ExpiresAt: future,
					CreatedBy: uuid.New(),
				})
				require.NoError(t, err)
			},
			wantErr: true,
			errMsg:  "invite already exists",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewInviteStore()
			
			if tc.setup != nil {
				tc.setup(store)
			}
			
			err := store.Create(ctx, tc.invite)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify it was stored
				stored, err := store.Get(ctx, tc.invite.ID)
				require.NoError(t, err)
				assert.Equal(t, tc.invite.ID, stored.ID)
				assert.Equal(t, tc.invite.Email, stored.Email)
				assert.Equal(t, tc.invite.Roles, stored.Roles)
				assert.Equal(t, tc.invite.TokenHash, stored.TokenHash)
			}
		})
	}
}

func TestInviteStore_Get(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	inviteID := uuid.New()
	createdBy := uuid.New()
	future := time.Now().Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(*InviteStore) *domain.Invite
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/found_invite",
			id:   inviteID,
			setup: func(store *InviteStore) *domain.Invite {
				invite := &domain.Invite{
					ID:        inviteID,
					Email:     "user@example.com",
					Roles:     []string{"user", "viewer"},
					TokenHash: []byte("token-hash"),
					ExpiresAt: future,
					Attempts:  0,
					CreatedBy: createdBy,
				}
				err := store.Create(ctx, invite)
				require.NoError(t, err)
				return invite
			},
			wantErr: false,
		},
		{
			name: "ok/redeemed_invite",
			id:   inviteID,
			setup: func(store *InviteStore) *domain.Invite {
				invite := &domain.Invite{
					ID:        inviteID,
					Email:     "redeemed@example.com",
					Roles:     []string{"admin"},
					TokenHash: []byte("redeemed-hash"),
					ExpiresAt: future,
					Attempts:  3,
					CreatedBy: createdBy,
				}
				err := store.Create(ctx, invite)
				require.NoError(t, err)
				
				// Mark as redeemed
				redeemedAt := time.Now()
				err = store.Redeem(ctx, inviteID, redeemedAt)
				require.NoError(t, err)
				
				// Update expected with redeemed time
				invite.RedeemedAt = &redeemedAt
				return invite
			},
			wantErr: false,
		},
		{
			name: "error/not_found",
			id:   uuid.New(),
			setup: func(store *InviteStore) *domain.Invite {
				// Create different invite
				err := store.Create(ctx, &domain.Invite{
					ID:        uuid.New(),
					Email:     "other@example.com",
					Roles:     []string{"user"},
					TokenHash: []byte("other-hash"),
					ExpiresAt: future,
					CreatedBy: createdBy,
				})
				require.NoError(t, err)
				return nil
			},
			wantErr: true,
			errMsg:  "invite not found",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewInviteStore()
			
			expected := tc.setup(store)
			
			actual, err := store.Get(ctx, tc.id)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, actual)
			} else {
				require.NoError(t, err)
				require.NotNil(t, actual)
				assert.Equal(t, expected.ID, actual.ID)
				assert.Equal(t, expected.Email, actual.Email)
				assert.Equal(t, expected.Roles, actual.Roles)
				assert.Equal(t, expected.TokenHash, actual.TokenHash)
			}
		})
	}
}

// Note: GetByHash is not implemented in the current InviteStore interface
// This test would be relevant if GetByHash method existed

func TestInviteStore_IncrementAttempts(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	inviteID := uuid.New()
	createdBy := uuid.New()
	future := time.Now().Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(*InviteStore) int
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/increment_from_zero",
			id:   inviteID,
			setup: func(store *InviteStore) int {
				err := store.Create(ctx, &domain.Invite{
					ID:        inviteID,
					Email:     "user@example.com",
					Roles:     []string{"user"},
					TokenHash: []byte("token-hash"),
					ExpiresAt: future,
					Attempts:  0,
					CreatedBy: createdBy,
				})
				require.NoError(t, err)
				return 0
			},
			wantErr: false,
		},
		{
			name: "ok/increment_multiple_times",
			id:   inviteID,
			setup: func(store *InviteStore) int {
				err := store.Create(ctx, &domain.Invite{
					ID:        inviteID,
					Email:     "user@example.com",
					Roles:     []string{"user"},
					TokenHash: []byte("token-hash"),
					ExpiresAt: future,
					Attempts:  5,
					CreatedBy: createdBy,
				})
				require.NoError(t, err)
				return 5
			},
			wantErr: false,
		},
		{
			name: "error/not_found",
			id:   uuid.New(),
			setup: func(store *InviteStore) int {
				return 0
			},
			wantErr: true,
			errMsg:  "invite not found or already redeemed",
		},
		{
			name: "error/already_redeemed",
			id:   inviteID,
			setup: func(store *InviteStore) int {
				err := store.Create(ctx, &domain.Invite{
					ID:        inviteID,
					Email:     "redeemed@example.com",
					Roles:     []string{"user"},
					TokenHash: []byte("token-hash"),
					ExpiresAt: future,
					Attempts:  2,
					CreatedBy: createdBy,
				})
				require.NoError(t, err)
				
				// Redeem it
				err = store.Redeem(ctx, inviteID, time.Now())
				require.NoError(t, err)
				
				return 2
			},
			wantErr: true,
			errMsg:  "invite not found or already redeemed",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewInviteStore()
			
			originalAttempts := tc.setup(store)
			
			err := store.IncrementAttempts(ctx, tc.id)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify attempts increased
				invite, err := store.Get(ctx, tc.id)
				require.NoError(t, err)
				assert.Equal(t, originalAttempts+1, invite.Attempts)
			}
		})
	}
}

func TestInviteStore_Redeem(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	inviteID := uuid.New()
	createdBy := uuid.New()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		id      uuid.UUID
		at      time.Time
		setup   func(*InviteStore)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/redeem_invite",
			id:   inviteID,
			at:   now,
			setup: func(store *InviteStore) {
				err := store.Create(ctx, &domain.Invite{
					ID:        inviteID,
					Email:     "user@example.com",
					Roles:     []string{"user"},
					TokenHash: []byte("token-hash"),
					ExpiresAt: future,
					Attempts:  3,
					CreatedBy: createdBy,
				})
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name: "error/not_found",
			id:   uuid.New(),
			at:   now,
			setup: func(store *InviteStore) {
				// No setup
			},
			wantErr: true,
			errMsg:  "invite not found or already redeemed",
		},
		{
			name: "error/already_redeemed",
			id:   inviteID,
			at:   now,
			setup: func(store *InviteStore) {
				err := store.Create(ctx, &domain.Invite{
					ID:        inviteID,
					Email:     "already@example.com",
					Roles:     []string{"user"},
					TokenHash: []byte("token-hash"),
					ExpiresAt: future,
					Attempts:  1,
					CreatedBy: createdBy,
				})
				require.NoError(t, err)
				
				// Redeem it once
				err = store.Redeem(ctx, inviteID, now.Add(-1*time.Hour))
				require.NoError(t, err)
			},
			wantErr: true,
			errMsg:  "invite not found or already redeemed",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewInviteStore()
			
			tc.setup(store)
			
			err := store.Redeem(ctx, tc.id, tc.at)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify it was redeemed
				invite, err := store.Get(ctx, tc.id)
				require.NoError(t, err)
				assert.NotNil(t, invite.RedeemedAt)
				assert.Equal(t, tc.at, *invite.RedeemedAt)
			}
		})
	}
}

func TestInviteStore_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	store := NewInviteStore()
	
	t.Run("concurrent_creates", func(t *testing.T) {
		t.Parallel()
		
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		errors := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				invite := &domain.Invite{
					ID:        uuid.New(),
					Email:     string(rune('a'+idx)) + "@example.com",
					Roles:     []string{"user"},
					TokenHash: []byte(string(rune('a' + idx))),
					ExpiresAt: time.Now().Add(24 * time.Hour),
					CreatedBy: uuid.New(),
				}
				
				if err := store.Create(ctx, invite); err != nil {
					errors <- err
				}
			}(i)
		}
		
		wg.Wait()
		close(errors)
		
		// Check for errors
		for err := range errors {
			assert.NoError(t, err)
		}
	})
	
	t.Run("concurrent_reads", func(t *testing.T) {
		t.Parallel()
		
		// Create an invite first
		inviteID := uuid.New()
		err := store.Create(ctx, &domain.Invite{
			ID:        inviteID,
			Email:     "read-test@example.com",
			Roles:     []string{"user"},
			TokenHash: []byte("read-test"),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedBy: uuid.New(),
		})
		require.NoError(t, err)
		
		const numGoroutines = 20
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				invite, err := store.Get(ctx, inviteID)
				assert.NoError(t, err)
				assert.NotNil(t, invite)
			}()
		}
		
		wg.Wait()
	})
	
	t.Run("concurrent_increments", func(t *testing.T) {
		t.Parallel()
		
		// Create an invite
		inviteID := uuid.New()
		err := store.Create(ctx, &domain.Invite{
			ID:        inviteID,
			Email:     "increment-test@example.com",
			Roles:     []string{"user"},
			TokenHash: []byte("increment-test"),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Attempts:  0,
			CreatedBy: uuid.New(),
		})
		require.NoError(t, err)
		
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				err := store.IncrementAttempts(ctx, inviteID)
				assert.NoError(t, err)
			}()
		}
		
		wg.Wait()
		
		// Verify final count
		invite, err := store.Get(ctx, inviteID)
		require.NoError(t, err)
		assert.Equal(t, numGoroutines, invite.Attempts)
	})
}

func TestInviteStore_DataIsolation(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("separate_stores_are_isolated", func(t *testing.T) {
		t.Parallel()
		
		store1 := NewInviteStore()
		store2 := NewInviteStore()
		
		// Create invite in store1
		inviteID := uuid.New()
		err := store1.Create(ctx, &domain.Invite{
			ID:        inviteID,
			Email:     "isolated@example.com",
			Roles:     []string{"user"},
			TokenHash: []byte("isolated-hash"),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedBy: uuid.New(),
		})
		require.NoError(t, err)
		
		// Should not exist in store2
		_, err = store2.Get(ctx, inviteID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invite not found")
	})
	
	t.Run("modifications_dont_affect_returned_objects", func(t *testing.T) {
		t.Parallel()
		
		store := NewInviteStore()
		
		// Create invite
		inviteID := uuid.New()
		original := &domain.Invite{
			ID:        inviteID,
			Email:     "test@example.com",
			Roles:     []string{"user", "admin"},
			TokenHash: []byte("original-hash"),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Attempts:  5,
			CreatedBy: uuid.New(),
		}
		err := store.Create(ctx, original)
		require.NoError(t, err)
		
		// Get invite
		retrieved, err := store.Get(ctx, inviteID)
		require.NoError(t, err)
		
		// Modify retrieved invite
		retrieved.Email = "modified@example.com"
		retrieved.Roles = []string{"superadmin"}
		retrieved.TokenHash = []byte("modified-hash")
		retrieved.Attempts = 999
		now := time.Now()
		retrieved.RedeemedAt = &now
		
		// Get invite again - should be unchanged
		unchanged, err := store.Get(ctx, inviteID)
		require.NoError(t, err)
		
		assert.Equal(t, "test@example.com", unchanged.Email)
		assert.Equal(t, []string{"user", "admin"}, unchanged.Roles)
		assert.Equal(t, []byte("original-hash"), unchanged.TokenHash)
		assert.Equal(t, 5, unchanged.Attempts)
		assert.Nil(t, unchanged.RedeemedAt)
	})
}
package inmemory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRefreshStore(t *testing.T) {
	t.Parallel()
	
	store := NewRefreshStore()
	assert.NotNil(t, store)
}

func TestRefreshStore_CreateFamily(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	
	tests := []struct {
		name           string
		userID         uuid.UUID
		sessionVersion int64
		tokenHash      []byte
		expiresAt      time.Time
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "ok/create_family",
			userID:         userID,
			sessionVersion: 1,
			tokenHash:      []byte("token-hash-123"),
			expiresAt:      time.Now().Add(24 * time.Hour),
			wantErr:        false,
		},
		{
			name:           "ok/high_session_version",
			userID:         userID,
			sessionVersion: 999,
			tokenHash:      []byte("token-hash-456"),
			expiresAt:      time.Now().Add(48 * time.Hour),
			wantErr:        false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRefreshStore()
			
			familyID, tokenID, err := store.CreateFamily(ctx, tc.userID, tc.sessionVersion, tc.tokenHash, tc.expiresAt)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Equal(t, uuid.Nil, familyID)
				assert.Equal(t, uuid.Nil, tokenID)
			} else {
				require.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, familyID)
				assert.NotEqual(t, uuid.Nil, tokenID)
				
				// Verify token was created
				token, err := store.GetByID(ctx, tokenID)
				require.NoError(t, err)
				assert.Equal(t, tokenID, token.ID)
				assert.Equal(t, familyID, token.FamilyID)
				assert.Equal(t, tc.userID, token.UserID)
				assert.Equal(t, tc.sessionVersion, token.SessionVersion)
				assert.Equal(t, tc.tokenHash, token.Hash)
			}
		})
	}
}

func TestRefreshStore_Rotate(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	
	tests := []struct {
		name        string
		setup       func(*RefreshStore) uuid.UUID
		newHash     []byte
		rotatedAt   time.Time
		expiresAt   time.Time
		wantErr     bool
		errMsg      string
	}{
		{
			name: "ok/rotate_token",
			setup: func(store *RefreshStore) uuid.UUID {
				_, tokenID, err := store.CreateFamily(ctx, userID, 1, []byte("old-hash"), future)
				require.NoError(t, err)
				return tokenID
			},
			newHash:   []byte("new-hash"),
			rotatedAt: now,
			expiresAt: future,
			wantErr:   false,
		},
		{
			name: "error/token_not_found",
			setup: func(store *RefreshStore) uuid.UUID {
				return uuid.New() // Non-existent token
			},
			newHash:   []byte("new-hash"),
			rotatedAt: now,
			expiresAt: future,
			wantErr:   true,
			errMsg:    "token not found",
		},
		{
			name: "error/already_rotated",
			setup: func(store *RefreshStore) uuid.UUID {
				_, tokenID, err := store.CreateFamily(ctx, userID, 1, []byte("old-hash"), future)
				require.NoError(t, err)
				
				// Rotate it once
				_, _, err = store.Rotate(ctx, tokenID, []byte("rotated-hash"), now.Add(-1*time.Hour), future)
				require.NoError(t, err)
				
				return tokenID
			},
			newHash:   []byte("new-hash-2"),
			rotatedAt: now,
			expiresAt: future,
			wantErr:   true,
			errMsg:    "token already rotated",
		},
		{
			name: "error/revoked_token",
			setup: func(store *RefreshStore) uuid.UUID {
				_, tokenID, err := store.CreateFamily(ctx, userID, 1, []byte("revoked-hash"), future)
				require.NoError(t, err)
				
				// Revoke it
				err = store.RevokeToken(ctx, tokenID, "test revocation", now)
				require.NoError(t, err)
				
				return tokenID
			},
			newHash:   []byte("new-hash"),
			rotatedAt: now,
			expiresAt: future,
			wantErr:   true,
			errMsg:    "token revoked",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRefreshStore()
			
			prevTokenID := tc.setup(store)
			
			newTokenID, familyID, err := store.Rotate(ctx, prevTokenID, tc.newHash, tc.rotatedAt, tc.expiresAt)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Equal(t, uuid.Nil, newTokenID)
				assert.Equal(t, uuid.Nil, familyID)
			} else {
				require.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, newTokenID)
				assert.NotEqual(t, uuid.Nil, familyID)
				assert.NotEqual(t, prevTokenID, newTokenID)
				
				// Verify old token is marked as rotated
				oldToken, err := store.GetByID(ctx, prevTokenID)
				require.NoError(t, err)
				assert.NotNil(t, oldToken.RotatedAt)
				assert.Equal(t, tc.rotatedAt, *oldToken.RotatedAt)
				
				// Verify new token exists
				newToken, err := store.GetByID(ctx, newTokenID)
				require.NoError(t, err)
				assert.Equal(t, newTokenID, newToken.ID)
				assert.Equal(t, familyID, newToken.FamilyID)
				assert.Equal(t, tc.newHash, newToken.Hash)
			}
		})
	}
}

func TestRefreshStore_GetByID(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	future := time.Now().Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		setup   func(*RefreshStore) uuid.UUID
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/found_token",
			setup: func(store *RefreshStore) uuid.UUID {
				_, tokenID, err := store.CreateFamily(ctx, userID, 1, []byte("hash"), future)
				require.NoError(t, err)
				return tokenID
			},
			wantErr: false,
		},
		{
			name: "error/not_found",
			setup: func(store *RefreshStore) uuid.UUID {
				return uuid.New()
			},
			wantErr: true,
			errMsg:  "token not found",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRefreshStore()
			
			tokenID := tc.setup(store)
			
			token, err := store.GetByID(ctx, tokenID)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, token)
			} else {
				require.NoError(t, err)
				require.NotNil(t, token)
				assert.Equal(t, tokenID, token.ID)
			}
		})
	}
}

func TestRefreshStore_GetByHash(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	future := time.Now().Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		hash    []byte
		setup   func(*RefreshStore)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/found_by_hash",
			hash: []byte("find-me"),
			setup: func(store *RefreshStore) {
				_, _, err := store.CreateFamily(ctx, userID, 1, []byte("find-me"), future)
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name: "error/not_found",
			hash: []byte("not-found"),
			setup: func(store *RefreshStore) {
				// Create a different token
				_, _, err := store.CreateFamily(ctx, userID, 1, []byte("different"), future)
				require.NoError(t, err)
			},
			wantErr: true,
			errMsg:  "token not found",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRefreshStore()
			
			tc.setup(store)
			
			token, err := store.GetByHash(ctx, tc.hash)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, token)
			} else {
				require.NoError(t, err)
				require.NotNil(t, token)
				assert.Equal(t, tc.hash, token.Hash)
			}
		})
	}
}

func TestRefreshStore_RevokeToken(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		setup   func(*RefreshStore) uuid.UUID
		reason  string
		at      time.Time
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/revoke_token",
			setup: func(store *RefreshStore) uuid.UUID {
				_, tokenID, err := store.CreateFamily(ctx, userID, 1, []byte("hash"), future)
				require.NoError(t, err)
				return tokenID
			},
			reason:  "suspicious activity",
			at:      now,
			wantErr: false,
		},
		{
			name: "error/not_found",
			setup: func(store *RefreshStore) uuid.UUID {
				return uuid.New()
			},
			reason:  "test",
			at:      now,
			wantErr: true,
			errMsg:  "token not found",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRefreshStore()
			
			tokenID := tc.setup(store)
			
			err := store.RevokeToken(ctx, tokenID, tc.reason, tc.at)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify token is revoked
				token, err := store.GetByID(ctx, tokenID)
				require.NoError(t, err)
				assert.NotNil(t, token.RevokedAt)
				assert.Equal(t, tc.at, *token.RevokedAt)
				assert.NotNil(t, token.Reason)
				assert.Equal(t, tc.reason, *token.Reason)
			}
		})
	}
}

func TestRefreshStore_RevokeFamily(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	
	tests := []struct {
		name     string
		setup    func(*RefreshStore) (uuid.UUID, []uuid.UUID)
		reason   string
		at       time.Time
		wantErr  bool
		errMsg   string
	}{
		{
			name: "ok/revoke_entire_family",
			setup: func(store *RefreshStore) (uuid.UUID, []uuid.UUID) {
				// Create initial token
				familyID, tokenID1, err := store.CreateFamily(ctx, userID, 1, []byte("hash1"), future)
				require.NoError(t, err)
				
				// Rotate to create more tokens in family
				tokenID2, _, err := store.Rotate(ctx, tokenID1, []byte("hash2"), now.Add(-1*time.Hour), future)
				require.NoError(t, err)
				
				tokenID3, _, err := store.Rotate(ctx, tokenID2, []byte("hash3"), now.Add(-30*time.Minute), future)
				require.NoError(t, err)
				
				return familyID, []uuid.UUID{tokenID1, tokenID2, tokenID3}
			},
			reason:  "replay attack detected",
			at:      now,
			wantErr: false,
		},
		{
			name: "ok/empty_family",
			setup: func(store *RefreshStore) (uuid.UUID, []uuid.UUID) {
				return uuid.New(), []uuid.UUID{}
			},
			reason:  "test",
			at:      now,
			wantErr: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRefreshStore()
			
			familyID, tokenIDs := tc.setup(store)
			
			err := store.RevokeFamily(ctx, familyID, tc.reason, tc.at)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify all tokens in family are revoked
				for _, tokenID := range tokenIDs {
					token, err := store.GetByID(ctx, tokenID)
					require.NoError(t, err)
					assert.NotNil(t, token.RevokedAt)
					assert.Equal(t, tc.at, *token.RevokedAt)
					assert.NotNil(t, token.Reason)
					assert.Equal(t, tc.reason, *token.Reason)
				}
			}
		})
	}
}

func TestRefreshStore_RevokeUserTokens(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	now := time.Now()
	future := now.Add(24 * time.Hour)
	
	tests := []struct {
		name    string
		userID  uuid.UUID
		setup   func(*RefreshStore) []uuid.UUID
		reason  string
		at      time.Time
		wantErr bool
		errMsg  string
	}{
		{
			name:   "ok/revoke_all_user_tokens",
			userID: userID,
			setup: func(store *RefreshStore) []uuid.UUID {
				tokenIDs := []uuid.UUID{}
				
				// Create multiple tokens for the user
				for i := 0; i < 3; i++ {
					_, tokenID, err := store.CreateFamily(ctx, userID, int64(i+1), []byte(string(rune('a'+i))), future)
					require.NoError(t, err)
					tokenIDs = append(tokenIDs, tokenID)
				}
				
				// Create token for different user (should not be affected)
				_, _, err := store.CreateFamily(ctx, otherUserID, 1, []byte("other"), future)
				require.NoError(t, err)
				
				return tokenIDs
			},
			reason:  "logout all devices",
			at:      now,
			wantErr: false,
		},
		{
			name:   "ok/no_tokens",
			userID: uuid.New(),
			setup: func(store *RefreshStore) []uuid.UUID {
				return []uuid.UUID{}
			},
			reason:  "test",
			at:      now,
			wantErr: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRefreshStore()
			
			tokenIDs := tc.setup(store)
			
			err := store.RevokeUserTokens(ctx, tc.userID, tc.reason, tc.at)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify all user tokens are revoked
				for _, tokenID := range tokenIDs {
					token, err := store.GetByID(ctx, tokenID)
					require.NoError(t, err)
					assert.NotNil(t, token.RevokedAt)
					assert.Equal(t, tc.at, *token.RevokedAt)
					assert.NotNil(t, token.Reason)
					assert.Equal(t, tc.reason, *token.Reason)
				}
			}
		})
	}
}

func TestRefreshStore_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	store := NewRefreshStore()
	
	t.Run("concurrent_creates", func(t *testing.T) {
		t.Parallel()
		
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		errors := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				_, _, err := store.CreateFamily(
					ctx,
					uuid.New(),
					int64(idx),
					[]byte(string(rune('a'+idx))),
					time.Now().Add(24*time.Hour),
				)
				
				if err != nil {
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
		
		// Create a token first
		_, tokenID, err := store.CreateFamily(ctx, uuid.New(), 1, []byte("read-test"), time.Now().Add(24*time.Hour))
		require.NoError(t, err)
		
		const numGoroutines = 20
		var wg sync.WaitGroup
		wg.Add(numGoroutines * 2) // For both GetByID and GetByHash
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				token, err := store.GetByID(ctx, tokenID)
				assert.NoError(t, err)
				assert.NotNil(t, token)
			}()
			
			go func() {
				defer wg.Done()
				
				token, err := store.GetByHash(ctx, []byte("read-test"))
				assert.NoError(t, err)
				assert.NotNil(t, token)
			}()
		}
		
		wg.Wait()
	})
	
	t.Run("concurrent_rotations", func(t *testing.T) {
		t.Parallel()
		
		// Create initial tokens
		const numFamilies = 5
		tokenIDs := make([]uuid.UUID, numFamilies)
		
		for i := 0; i < numFamilies; i++ {
			_, tokenID, err := store.CreateFamily(
				ctx,
				uuid.New(),
				1,
				[]byte(string(rune('a'+i))),
				time.Now().Add(24*time.Hour),
			)
			require.NoError(t, err)
			tokenIDs[i] = tokenID
		}
		
		var wg sync.WaitGroup
		wg.Add(numFamilies)
		
		for i := 0; i < numFamilies; i++ {
			go func(idx int) {
				defer wg.Done()
				
				_, _, err := store.Rotate(
					ctx,
					tokenIDs[idx],
					[]byte(string(rune('z'-idx))),
					time.Now(),
					time.Now().Add(24*time.Hour),
				)
				assert.NoError(t, err)
			}(i)
		}
		
		wg.Wait()
	})
}

func TestRefreshStore_DataIsolation(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("separate_stores_are_isolated", func(t *testing.T) {
		t.Parallel()
		
		store1 := NewRefreshStore()
		store2 := NewRefreshStore()
		
		// Create token in store1
		_, tokenID, err := store1.CreateFamily(ctx, uuid.New(), 1, []byte("hash"), time.Now().Add(24*time.Hour))
		require.NoError(t, err)
		
		// Should not exist in store2
		_, err = store2.GetByID(ctx, tokenID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token not found")
		
		_, err = store2.GetByHash(ctx, []byte("hash"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token not found")
	})
	
	t.Run("modifications_dont_affect_returned_objects", func(t *testing.T) {
		t.Parallel()
		
		store := NewRefreshStore()
		
		// Create token
		_, tokenID, err := store.CreateFamily(
			ctx,
			uuid.New(),
			10,
			[]byte("original-hash"),
			time.Now().Add(24*time.Hour),
		)
		require.NoError(t, err)
		
		// Get token
		retrieved, err := store.GetByID(ctx, tokenID)
		require.NoError(t, err)
		
		// Modify retrieved token
		retrieved.Hash = []byte("modified-hash")
		retrieved.SessionVersion = 999
		now := time.Now()
		retrieved.RevokedAt = &now
		
		// Get token again - should be unchanged
		unchanged, err := store.GetByID(ctx, tokenID)
		require.NoError(t, err)
		
		assert.Equal(t, []byte("original-hash"), unchanged.Hash)
		assert.Equal(t, int64(10), unchanged.SessionVersion)
		assert.Nil(t, unchanged.RevokedAt)
	})
}
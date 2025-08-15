package inmemory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCredentialStore(t *testing.T) {
	t.Parallel()
	
	store := NewCredentialStore()
	assert.NotNil(t, store)
}

func TestCredentialStore_Add(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	
	tests := []struct {
		name    string
		cred    *domain.Credential
		setup   func(*CredentialStore)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/basic_credential",
			cred: &domain.Credential{
				ID:         []byte("cred-id-123"),
				UserID:     userID,
				PublicKey:  []byte("public-key"),
				AAGUID:     "aaguid-123",
				DeviceName: "YubiKey 5",
				RK:         true,
				Transports: []string{"usb"},
				SignCount:  0,
			},
			wantErr: false,
		},
		{
			name: "ok/multiple_transports",
			cred: &domain.Credential{
				ID:         []byte("cred-id-456"),
				UserID:     userID,
				PublicKey:  []byte("public-key-2"),
				AAGUID:     "aaguid-456",
				DeviceName: "TouchID",
				RK:         true,
				Transports: []string{"internal", "hybrid"},
				SignCount:  0,
			},
			wantErr: false,
		},
		{
			name: "error/duplicate_credential",
			cred: &domain.Credential{
				ID:         []byte("duplicate-id"),
				UserID:     userID,
				PublicKey:  []byte("public-key"),
				AAGUID:     "aaguid-dup",
				DeviceName: "Device",
				RK:         false,
			},
			setup: func(store *CredentialStore) {
				err := store.Add(ctx, &domain.Credential{
					ID:        []byte("duplicate-id"),
					UserID:    uuid.New(),
					PublicKey: []byte("other-key"),
				})
				require.NoError(t, err)
			},
			wantErr: true,
			errMsg:  "credential already exists",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewCredentialStore()
			
			if tc.setup != nil {
				tc.setup(store)
			}
			
			err := store.Add(ctx, tc.cred)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify it was stored
				stored, err := store.Get(ctx, tc.cred.ID)
				require.NoError(t, err)
				assert.Equal(t, tc.cred.ID, stored.ID)
				assert.Equal(t, tc.cred.UserID, stored.UserID)
				assert.Equal(t, tc.cred.PublicKey, stored.PublicKey)
			}
		})
	}
}

func TestCredentialStore_Get(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	
	tests := []struct {
		name    string
		id      []byte
		setup   func(*CredentialStore) *domain.Credential
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/found_credential",
			id:   []byte("cred-id"),
			setup: func(store *CredentialStore) *domain.Credential {
				cred := &domain.Credential{
					ID:         []byte("cred-id"),
					UserID:     userID,
					PublicKey:  []byte("public-key"),
					AAGUID:     "aaguid",
					DeviceName: "Device",
					RK:         true,
					SignCount:  42,
				}
				err := store.Add(ctx, cred)
				require.NoError(t, err)
				return cred
			},
			wantErr: false,
		},
		{
			name: "error/not_found",
			id:   []byte("not-found"),
			setup: func(store *CredentialStore) *domain.Credential {
				return nil
			},
			wantErr: true,
			errMsg:  "credential not found",
		},
		{
			name: "error/revoked_credential",
			id:   []byte("revoked-id"),
			setup: func(store *CredentialStore) *domain.Credential {
				cred := &domain.Credential{
					ID:        []byte("revoked-id"),
					UserID:    userID,
					PublicKey: []byte("public-key"),
				}
				err := store.Add(ctx, cred)
				require.NoError(t, err)
				err = store.Revoke(ctx, []byte("revoked-id"))
				require.NoError(t, err)
				return cred
			},
			wantErr: true,
			errMsg:  "credential revoked",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewCredentialStore()
			
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
				assert.Equal(t, expected.UserID, actual.UserID)
				assert.Equal(t, expected.PublicKey, actual.PublicKey)
				assert.Equal(t, expected.SignCount, actual.SignCount)
			}
		})
	}
}

func TestCredentialStore_GetByUser(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	otherUserID := uuid.New()
	
	tests := []struct {
		name     string
		userID   uuid.UUID
		setup    func(*CredentialStore) []*domain.Credential
		wantErr  bool
		errMsg   string
		validate func(*testing.T, []domain.Credential)
	}{
		{
			name:   "ok/multiple_credentials",
			userID: userID,
			setup: func(store *CredentialStore) []*domain.Credential {
				creds := []*domain.Credential{
					{
						ID:         []byte("cred-1"),
						UserID:     userID,
						PublicKey:  []byte("key-1"),
						DeviceName: "Device 1",
					},
					{
						ID:         []byte("cred-2"),
						UserID:     userID,
						PublicKey:  []byte("key-2"),
						DeviceName: "Device 2",
					},
					{
						ID:         []byte("cred-3"),
						UserID:     otherUserID,
						PublicKey:  []byte("key-3"),
						DeviceName: "Other Device",
					},
				}
				
				for _, cred := range creds {
					err := store.Add(ctx, cred)
					require.NoError(t, err)
				}
				
				return creds[:2] // Only first two belong to userID
			},
			wantErr: false,
			validate: func(t *testing.T, creds []domain.Credential) {
				assert.Len(t, creds, 2)
				for _, cred := range creds {
					assert.Equal(t, userID, cred.UserID)
				}
			},
		},
		{
			name:   "ok/no_credentials",
			userID: uuid.New(),
			setup: func(store *CredentialStore) []*domain.Credential {
				return []*domain.Credential{}
			},
			wantErr: false,
			validate: func(t *testing.T, creds []domain.Credential) {
				assert.Len(t, creds, 0)
			},
		},
		{
			name:   "ok/excludes_revoked",
			userID: userID,
			setup: func(store *CredentialStore) []*domain.Credential {
				creds := []*domain.Credential{
					{
						ID:        []byte("active-cred"),
						UserID:    userID,
						PublicKey: []byte("key-1"),
					},
					{
						ID:        []byte("revoked-cred"),
						UserID:    userID,
						PublicKey: []byte("key-2"),
					},
				}
				
				for _, cred := range creds {
					err := store.Add(ctx, cred)
					require.NoError(t, err)
				}
				
				// Revoke the second credential
				err := store.Revoke(ctx, []byte("revoked-cred"))
				require.NoError(t, err)
				
				return creds[:1] // Only first one should be returned
			},
			wantErr: false,
			validate: func(t *testing.T, creds []domain.Credential) {
				assert.Len(t, creds, 1)
				assert.Equal(t, []byte("active-cred"), creds[0].ID)
			},
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewCredentialStore()
			
			tc.setup(store)
			
			creds, err := store.GetByUser(ctx, tc.userID)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, creds)
			} else {
				require.NoError(t, err)
				require.NotNil(t, creds)
				
				if tc.validate != nil {
					tc.validate(t, creds)
				}
			}
		})
	}
}

func TestCredentialStore_UpdateOnAssertion(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	
	tests := []struct {
		name      string
		id        []byte
		signCount uint32
		setup     func(*CredentialStore) uint32
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "ok/update_sign_count",
			id:        []byte("cred-id"),
			signCount: 43,
			setup: func(store *CredentialStore) uint32 {
				cred := &domain.Credential{
					ID:        []byte("cred-id"),
					UserID:    userID,
					PublicKey: []byte("public-key"),
					SignCount: 42,
				}
				err := store.Add(ctx, cred)
				require.NoError(t, err)
				return 42
			},
			wantErr: false,
		},
		{
			name:      "error/not_found",
			id:        []byte("not-found"),
			signCount: 1,
			setup: func(store *CredentialStore) uint32 {
				return 0
			},
			wantErr: true,
			errMsg:  "credential not found",
		},
		{
			name:      "error/revoked_credential",
			id:        []byte("revoked-id"),
			signCount: 10,
			setup: func(store *CredentialStore) uint32 {
				cred := &domain.Credential{
					ID:        []byte("revoked-id"),
					UserID:    userID,
					PublicKey: []byte("public-key"),
					SignCount: 5,
				}
				err := store.Add(ctx, cred)
				require.NoError(t, err)
				err = store.Revoke(ctx, []byte("revoked-id"))
				require.NoError(t, err)
				return 5
			},
			wantErr: true,
			errMsg:  "credential revoked",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewCredentialStore()
			
			originalCount := tc.setup(store)
			
			err := store.UpdateOnAssertion(ctx, tc.id, tc.signCount, time.Now())
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify the update
				cred, err := store.Get(ctx, tc.id)
				require.NoError(t, err)
				assert.Equal(t, tc.signCount, cred.SignCount)
				assert.NotZero(t, cred.LastUsedAt)
				assert.Greater(t, cred.SignCount, originalCount)
			}
		})
	}
}

func TestCredentialStore_Revoke(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	
	tests := []struct {
		name    string
		id      []byte
		setup   func(*CredentialStore)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/revoke_credential",
			id:   []byte("cred-id"),
			setup: func(store *CredentialStore) {
				cred := &domain.Credential{
					ID:        []byte("cred-id"),
					UserID:    userID,
					PublicKey: []byte("public-key"),
				}
				err := store.Add(ctx, cred)
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name: "error/not_found",
			id:   []byte("not-found"),
			setup: func(store *CredentialStore) {
				// No setup
			},
			wantErr: true,
			errMsg:  "credential not found",
		},
		{
			name: "ok/already_revoked",
			id:   []byte("already-revoked"),
			setup: func(store *CredentialStore) {
				cred := &domain.Credential{
					ID:        []byte("already-revoked"),
					UserID:    userID,
					PublicKey: []byte("public-key"),
				}
				err := store.Add(ctx, cred)
				require.NoError(t, err)
				err = store.Revoke(ctx, []byte("already-revoked"))
				require.NoError(t, err)
			},
			wantErr: false, // Revoking already revoked credential is idempotent
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewCredentialStore()
			
			tc.setup(store)
			
			err := store.Revoke(ctx, tc.id)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify it's revoked (Get should fail)
				_, err = store.Get(ctx, tc.id)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "credential revoked")
			}
		})
	}
}

func TestCredentialStore_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	store := NewCredentialStore()
	
	t.Run("concurrent_adds", func(t *testing.T) {
		t.Parallel()
		
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		errors := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				cred := &domain.Credential{
					ID:        []byte(string(rune('a' + idx))),
					UserID:    uuid.New(),
					PublicKey: []byte("key"),
				}
				
				if err := store.Add(ctx, cred); err != nil {
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
		
		// Add a credential first
		cred := &domain.Credential{
			ID:        []byte("read-test"),
			UserID:    uuid.New(),
			PublicKey: []byte("key"),
		}
		err := store.Add(ctx, cred)
		require.NoError(t, err)
		
		const numGoroutines = 20
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				found, err := store.Get(ctx, []byte("read-test"))
				assert.NoError(t, err)
				assert.NotNil(t, found)
			}()
		}
		
		wg.Wait()
	})
	
	t.Run("concurrent_updates", func(t *testing.T) {
		t.Parallel()
		
		// Add a credential first
		userID := uuid.New()
		cred := &domain.Credential{
			ID:        []byte("update-test"),
			UserID:    userID,
			PublicKey: []byte("key"),
			SignCount: 0,
		}
		err := store.Add(ctx, cred)
		require.NoError(t, err)
		
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				err := store.UpdateOnAssertion(ctx, []byte("update-test"), uint32(idx+1), time.Now())
				assert.NoError(t, err)
			}(i)
		}
		
		wg.Wait()
		
		// Verify final state
		final, err := store.Get(ctx, []byte("update-test"))
		require.NoError(t, err)
		assert.Greater(t, final.SignCount, uint32(0))
	})
}

func TestCredentialStore_DataIsolation(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("separate_stores_are_isolated", func(t *testing.T) {
		t.Parallel()
		
		store1 := NewCredentialStore()
		store2 := NewCredentialStore()
		
		// Add credential to store1
		cred := &domain.Credential{
			ID:        []byte("isolated-cred"),
			UserID:    uuid.New(),
			PublicKey: []byte("key"),
		}
		err := store1.Add(ctx, cred)
		require.NoError(t, err)
		
		// Should not exist in store2
		_, err = store2.Get(ctx, []byte("isolated-cred"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "credential not found")
	})
	
	t.Run("modifications_dont_affect_returned_objects", func(t *testing.T) {
		t.Parallel()
		
		store := NewCredentialStore()
		userID := uuid.New()
		
		// Add credential
		original := &domain.Credential{
			ID:         []byte("test-cred"),
			UserID:     userID,
			PublicKey:  []byte("original-key"),
			DeviceName: "Original Device",
			SignCount:  10,
		}
		err := store.Add(ctx, original)
		require.NoError(t, err)
		
		// Get credential
		retrieved, err := store.Get(ctx, []byte("test-cred"))
		require.NoError(t, err)
		
		// Modify retrieved credential
		retrieved.PublicKey = []byte("modified-key")
		retrieved.DeviceName = "Modified Device"
		retrieved.SignCount = 999
		
		// Get credential again - should be unchanged
		unchanged, err := store.Get(ctx, []byte("test-cred"))
		require.NoError(t, err)
		
		assert.Equal(t, []byte("original-key"), unchanged.PublicKey)
		assert.Equal(t, "Original Device", unchanged.DeviceName)
		assert.Equal(t, uint32(10), unchanged.SignCount)
	})
}
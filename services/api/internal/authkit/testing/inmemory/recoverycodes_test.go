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

func TestNewRecoveryCodeStore(t *testing.T) {
	t.Parallel()
	
	store := NewRecoveryCodeStore()
	assert.NotNil(t, store)
}

func TestRecoveryCodeStore_ReplaceCodes(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	
	tests := []struct {
		name       string
		userID     uuid.UUID
		codeHashes [][]byte
		setup      func(*RecoveryCodeStore)
		wantErr    bool
		errMsg     string
		validate   func(*testing.T, *RecoveryCodeStore)
	}{
		{
			name:       "ok/new_codes",
			userID:     userID,
			codeHashes: [][]byte{[]byte("code1"), []byte("code2"), []byte("code3")},
			wantErr:    false,
			validate: func(t *testing.T, store *RecoveryCodeStore) {
				codes, err := store.GetByUserID(ctx, userID)
				require.NoError(t, err)
				assert.Len(t, codes, 3)
				
				// Verify all codes exist and have correct properties
				for i, code := range codes {
					assert.Equal(t, userID, code.UserID)
					assert.Equal(t, []byte("code"+string(rune('1'+i))), code.Hash)
					assert.NotZero(t, code.CreatedAt)
					assert.Nil(t, code.UsedAt)
				}
			},
		},
		{
			name:       "ok/replace_existing",
			userID:     userID,
			codeHashes: [][]byte{[]byte("new-code1"), []byte("new-code2")},
			setup: func(store *RecoveryCodeStore) {
				// Add existing codes first
				err := store.ReplaceCodes(ctx, userID, [][]byte{[]byte("old-code1"), []byte("old-code2"), []byte("old-code3")})
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, store *RecoveryCodeStore) {
				codes, err := store.GetByUserID(ctx, userID)
				require.NoError(t, err)
				assert.Len(t, codes, 2)
				
				// Verify old codes are gone
				for _, code := range codes {
					assert.NotEqual(t, []byte("old-code1"), code.Hash)
					assert.NotEqual(t, []byte("old-code2"), code.Hash)
					assert.NotEqual(t, []byte("old-code3"), code.Hash)
				}
			},
		},
		{
			name:       "ok/empty_codes",
			userID:     userID,
			codeHashes: [][]byte{},
			wantErr:    false,
			validate: func(t *testing.T, store *RecoveryCodeStore) {
				codes, err := store.GetByUserID(ctx, userID)
				require.NoError(t, err)
				assert.Len(t, codes, 0)
			},
		},
		{
			name:       "ok/single_code",
			userID:     userID,
			codeHashes: [][]byte{[]byte("single-code")},
			wantErr:    false,
			validate: func(t *testing.T, store *RecoveryCodeStore) {
				codes, err := store.GetByUserID(ctx, userID)
				require.NoError(t, err)
				assert.Len(t, codes, 1)
				assert.Equal(t, []byte("single-code"), codes[0].Hash)
			},
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRecoveryCodeStore()
			
			if tc.setup != nil {
				tc.setup(store)
			}
			
			err := store.ReplaceCodes(ctx, tc.userID, tc.codeHashes)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				if tc.validate != nil {
					tc.validate(t, store)
				}
			}
		})
	}
}

func TestRecoveryCodeStore_Consume(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name     string
		userID   uuid.UUID
		codeHash []byte
		at       time.Time
		setup    func(*RecoveryCodeStore)
		wantUsed bool
		wantErr  bool
		errMsg   string
		validate func(*testing.T, *RecoveryCodeStore)
	}{
		{
			name:     "ok/consume_valid_code",
			userID:   userID,
			codeHash: []byte("valid-code"),
			at:       now,
			setup: func(store *RecoveryCodeStore) {
				err := store.ReplaceCodes(ctx, userID, [][]byte{
					[]byte("valid-code"),
					[]byte("other-code"),
				})
				require.NoError(t, err)
			},
			wantUsed: true,
			wantErr:  false,
			validate: func(t *testing.T, store *RecoveryCodeStore) {
				// Verify code is marked as used
				code, err := store.GetByHash(ctx, userID, []byte("valid-code"))
				require.NoError(t, err)
				require.NotNil(t, code)
				assert.NotNil(t, code.UsedAt)
				assert.Equal(t, now, *code.UsedAt)
				
				// Verify unused count decreased
				count, err := store.CountUnused(ctx, userID)
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name:     "ok/code_not_found",
			userID:   userID,
			codeHash: []byte("nonexistent-code"),
			at:       now,
			setup: func(store *RecoveryCodeStore) {
				err := store.ReplaceCodes(ctx, userID, [][]byte{
					[]byte("different-code"),
				})
				require.NoError(t, err)
			},
			wantUsed: false,
			wantErr:  false,
		},
		{
			name:     "ok/user_has_no_codes",
			userID:   uuid.New(),
			codeHash: []byte("any-code"),
			at:       now,
			setup: func(store *RecoveryCodeStore) {
				// Setup different user's codes
				err := store.ReplaceCodes(ctx, userID, [][]byte{
					[]byte("other-user-code"),
				})
				require.NoError(t, err)
			},
			wantUsed: false,
			wantErr:  false,
		},
		{
			name:     "ok/code_already_used",
			userID:   userID,
			codeHash: []byte("used-code"),
			at:       now,
			setup: func(store *RecoveryCodeStore) {
				err := store.ReplaceCodes(ctx, userID, [][]byte{
					[]byte("used-code"),
					[]byte("unused-code"),
				})
				require.NoError(t, err)
				
				// Use the code once
				used, err := store.Consume(ctx, userID, []byte("used-code"), now.Add(-1*time.Hour))
				require.NoError(t, err)
				assert.True(t, used)
			},
			wantUsed: false,
			wantErr:  false,
			validate: func(t *testing.T, store *RecoveryCodeStore) {
				// Verify code is still marked as used from before
				code, err := store.GetByHash(ctx, userID, []byte("used-code"))
				require.NoError(t, err)
				require.NotNil(t, code)
				assert.NotNil(t, code.UsedAt)
				assert.Equal(t, now.Add(-1*time.Hour), *code.UsedAt) // Original use time, not current
			},
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRecoveryCodeStore()
			
			if tc.setup != nil {
				tc.setup(store)
			}
			
			used, err := store.Consume(ctx, tc.userID, tc.codeHash, tc.at)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantUsed, used)
				
				if tc.validate != nil {
					tc.validate(t, store)
				}
			}
		})
	}
}

func TestRecoveryCodeStore_List(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name     string
		userID   uuid.UUID
		setup    func(*RecoveryCodeStore)
		wantErr  bool
		errMsg   string
		validate func(*testing.T, [][]byte)
	}{
		{
			name:   "ok/all_unused_codes",
			userID: userID,
			setup: func(store *RecoveryCodeStore) {
				err := store.ReplaceCodes(ctx, userID, [][]byte{
					[]byte("code1"),
					[]byte("code2"),
					[]byte("code3"),
				})
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, hashes [][]byte) {
				assert.Len(t, hashes, 3)
				expectedHashes := [][]byte{[]byte("code1"), []byte("code2"), []byte("code3")}
				for _, expected := range expectedHashes {
					assert.Contains(t, hashes, expected)
				}
			},
		},
		{
			name:   "ok/excludes_used_codes",
			userID: userID,
			setup: func(store *RecoveryCodeStore) {
				err := store.ReplaceCodes(ctx, userID, [][]byte{
					[]byte("unused1"),
					[]byte("used1"),
					[]byte("unused2"),
					[]byte("used2"),
				})
				require.NoError(t, err)
				
				// Use some codes
				used, err := store.Consume(ctx, userID, []byte("used1"), now)
				require.NoError(t, err)
				assert.True(t, used)
				
				used, err = store.Consume(ctx, userID, []byte("used2"), now)
				require.NoError(t, err)
				assert.True(t, used)
			},
			wantErr: false,
			validate: func(t *testing.T, hashes [][]byte) {
				assert.Len(t, hashes, 2)
				assert.Contains(t, hashes, []byte("unused1"))
				assert.Contains(t, hashes, []byte("unused2"))
				assert.NotContains(t, hashes, []byte("used1"))
				assert.NotContains(t, hashes, []byte("used2"))
			},
		},
		{
			name:   "ok/no_codes",
			userID: uuid.New(),
			setup: func(store *RecoveryCodeStore) {
				// No codes for this user
			},
			wantErr: false,
			validate: func(t *testing.T, hashes [][]byte) {
				assert.Len(t, hashes, 0)
			},
		},
		{
			name:   "ok/all_codes_used",
			userID: userID,
			setup: func(store *RecoveryCodeStore) {
				err := store.ReplaceCodes(ctx, userID, [][]byte{
					[]byte("code1"),
					[]byte("code2"),
				})
				require.NoError(t, err)
				
				// Use all codes
				used, err := store.Consume(ctx, userID, []byte("code1"), now)
				require.NoError(t, err)
				assert.True(t, used)
				
				used, err = store.Consume(ctx, userID, []byte("code2"), now)
				require.NoError(t, err)
				assert.True(t, used)
			},
			wantErr: false,
			validate: func(t *testing.T, hashes [][]byte) {
				assert.Len(t, hashes, 0)
			},
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRecoveryCodeStore()
			
			if tc.setup != nil {
				tc.setup(store)
			}
			
			hashes, err := store.List(ctx, tc.userID)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, hashes)
			} else {
				require.NoError(t, err)
				require.NotNil(t, hashes)
				
				if tc.validate != nil {
					tc.validate(t, hashes)
				}
			}
		})
	}
}

func TestRecoveryCodeStore_CountUnused(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()
	
	tests := []struct {
		name      string
		userID    uuid.UUID
		setup     func(*RecoveryCodeStore)
		wantCount int
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "ok/all_unused",
			userID: userID,
			setup: func(store *RecoveryCodeStore) {
				err := store.ReplaceCodes(ctx, userID, [][]byte{
					[]byte("code1"),
					[]byte("code2"),
					[]byte("code3"),
					[]byte("code4"),
					[]byte("code5"),
				})
				require.NoError(t, err)
			},
			wantCount: 5,
			wantErr:   false,
		},
		{
			name:   "ok/some_used",
			userID: userID,
			setup: func(store *RecoveryCodeStore) {
				err := store.ReplaceCodes(ctx, userID, [][]byte{
					[]byte("unused1"),
					[]byte("unused2"),
					[]byte("used1"),
					[]byte("used2"),
				})
				require.NoError(t, err)
				
				// Use some codes
				used, err := store.Consume(ctx, userID, []byte("used1"), now)
				require.NoError(t, err)
				assert.True(t, used)
				
				used, err = store.Consume(ctx, userID, []byte("used2"), now)
				require.NoError(t, err)
				assert.True(t, used)
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "ok/no_codes",
			userID:    uuid.New(),
			setup:     func(store *RecoveryCodeStore) {},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:   "ok/all_used",
			userID: userID,
			setup: func(store *RecoveryCodeStore) {
				err := store.ReplaceCodes(ctx, userID, [][]byte{
					[]byte("code1"),
					[]byte("code2"),
				})
				require.NoError(t, err)
				
				// Use all codes
				used, err := store.Consume(ctx, userID, []byte("code1"), now)
				require.NoError(t, err)
				assert.True(t, used)
				
				used, err = store.Consume(ctx, userID, []byte("code2"), now)
				require.NoError(t, err)
				assert.True(t, used)
			},
			wantCount: 0,
			wantErr:   false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewRecoveryCodeStore()
			
			if tc.setup != nil {
				tc.setup(store)
			}
			
			count, err := store.CountUnused(ctx, tc.userID)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantCount, count)
			}
		})
	}
}

func TestRecoveryCodeStore_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	store := NewRecoveryCodeStore()
	
	t.Run("concurrent_replace_codes", func(t *testing.T) {
		t.Parallel()
		
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		errors := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				userID := uuid.New()
				codeHashes := [][]byte{
					[]byte("code1-" + string(rune('a'+idx))),
					[]byte("code2-" + string(rune('a'+idx))),
				}
				
				if err := store.ReplaceCodes(ctx, userID, codeHashes); err != nil {
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
	
	t.Run("concurrent_consume_same_user", func(t *testing.T) {
		t.Parallel()
		
		userID := uuid.New()
		
		// Create multiple codes
		codeHashes := make([][]byte, 10)
		for i := 0; i < 10; i++ {
			codeHashes[i] = []byte("code-" + string(rune('a'+i)))
		}
		
		err := store.ReplaceCodes(ctx, userID, codeHashes)
		require.NoError(t, err)
		
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		usedCodes := make(chan bool, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				codeHash := []byte("code-" + string(rune('a'+idx)))
				used, err := store.Consume(ctx, userID, codeHash, time.Now())
				assert.NoError(t, err)
				usedCodes <- used
			}(i)
		}
		
		wg.Wait()
		close(usedCodes)
		
		// All codes should have been successfully consumed
		successCount := 0
		for used := range usedCodes {
			if used {
				successCount++
			}
		}
		assert.Equal(t, 10, successCount)
		
		// Verify final count
		count, err := store.CountUnused(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
	
	t.Run("concurrent_reads", func(t *testing.T) {
		t.Parallel()
		
		userID := uuid.New()
		err := store.ReplaceCodes(ctx, userID, [][]byte{
			[]byte("read-test-1"),
			[]byte("read-test-2"),
		})
		require.NoError(t, err)
		
		const numGoroutines = 20
		var wg sync.WaitGroup
		wg.Add(numGoroutines * 2) // For both List and CountUnused
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				hashes, err := store.List(ctx, userID)
				assert.NoError(t, err)
				assert.Len(t, hashes, 2)
			}()
			
			go func() {
				defer wg.Done()
				
				count, err := store.CountUnused(ctx, userID)
				assert.NoError(t, err)
				assert.Equal(t, 2, count)
			}()
		}
		
		wg.Wait()
	})
}

func TestRecoveryCodeStore_DataIsolation(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("separate_stores_are_isolated", func(t *testing.T) {
		t.Parallel()
		
		store1 := NewRecoveryCodeStore()
		store2 := NewRecoveryCodeStore()
		
		userID := uuid.New()
		
		// Add codes to store1
		err := store1.ReplaceCodes(ctx, userID, [][]byte{
			[]byte("isolated-code1"),
			[]byte("isolated-code2"),
		})
		require.NoError(t, err)
		
		// Should not exist in store2
		count, err := store2.CountUnused(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		
		hashes, err := store2.List(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, hashes, 0)
	})
	
	t.Run("user_isolation", func(t *testing.T) {
		t.Parallel()
		
		store := NewRecoveryCodeStore()
		
		user1 := uuid.New()
		user2 := uuid.New()
		
		// Add codes for user1
		err := store.ReplaceCodes(ctx, user1, [][]byte{
			[]byte("user1-code1"),
			[]byte("user1-code2"),
		})
		require.NoError(t, err)
		
		// Add codes for user2
		err = store.ReplaceCodes(ctx, user2, [][]byte{
			[]byte("user2-code1"),
		})
		require.NoError(t, err)
		
		// User1 should only see their codes
		count1, err := store.CountUnused(ctx, user1)
		require.NoError(t, err)
		assert.Equal(t, 2, count1)
		
		hashes1, err := store.List(ctx, user1)
		require.NoError(t, err)
		assert.Len(t, hashes1, 2)
		
		// User2 should only see their codes
		count2, err := store.CountUnused(ctx, user2)
		require.NoError(t, err)
		assert.Equal(t, 1, count2)
		
		hashes2, err := store.List(ctx, user2)
		require.NoError(t, err)
		assert.Len(t, hashes2, 1)
		
		// User1 shouldn't be able to consume user2's codes
		used, err := store.Consume(ctx, user1, []byte("user2-code1"), time.Now())
		require.NoError(t, err)
		assert.False(t, used)
	})
	
	t.Run("modifications_dont_affect_returned_objects", func(t *testing.T) {
		t.Parallel()
		
		store := NewRecoveryCodeStore()
		userID := uuid.New()
		
		// Add codes
		originalHashes := [][]byte{[]byte("original1"), []byte("original2")}
		err := store.ReplaceCodes(ctx, userID, originalHashes)
		require.NoError(t, err)
		
		// Get hashes
		retrievedHashes, err := store.List(ctx, userID)
		require.NoError(t, err)
		
		// Modify returned hashes
		for i := range retrievedHashes {
			retrievedHashes[i] = []byte("modified")
		}
		
		// Get hashes again - should be unchanged
		unchangedHashes, err := store.List(ctx, userID)
		require.NoError(t, err)
		
		assert.Equal(t, originalHashes, unchangedHashes)
		assert.NotEqual(t, retrievedHashes, unchangedHashes)
	})
}
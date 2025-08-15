package inmemory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChallengeStore(t *testing.T) {
	t.Parallel()
	
	store := NewChallengeStore()
	assert.NotNil(t, store)
}

func TestChallengeStore_Set(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		key     string
		value   []byte
		ttl     time.Duration
		wantErr bool
		errMsg  string
	}{
		{
			name:    "ok/basic_set",
			key:     "test-key",
			value:   []byte("test-value"),
			ttl:     time.Minute,
			wantErr: false,
		},
		{
			name:    "ok/empty_value",
			key:     "empty-key",
			value:   []byte{},
			ttl:     time.Minute,
			wantErr: false,
		},
		{
			name:    "ok/short_ttl",
			key:     "short-key",
			value:   []byte("short-value"),
			ttl:     time.Millisecond,
			wantErr: false,
		},
		{
			name:    "ok/long_ttl",
			key:     "long-key",
			value:   []byte("long-value"),
			ttl:     time.Hour,
			wantErr: false,
		},
		{
			name:    "ok/overwrite_existing",
			key:     "overwrite-key",
			value:   []byte("new-value"),
			ttl:     time.Minute,
			wantErr: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewChallengeStore()
			
			// For overwrite test, set an initial value
			if tc.name == "ok/overwrite_existing" {
				err := store.Set(ctx, tc.key, []byte("old-value"), time.Minute)
				require.NoError(t, err)
			}
			
			err := store.Set(ctx, tc.key, tc.value, tc.ttl)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify the challenge was stored
				retrieved, err := store.Get(ctx, tc.key)
				require.NoError(t, err)
				assert.Equal(t, tc.value, retrieved)
			}
		})
	}
}

func TestChallengeStore_Get(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name     string
		key      string
		setup    func(*ChallengeStore)
		wantErr  bool
		errMsg   string
		validate func(*testing.T, []byte)
	}{
		{
			name: "ok/valid_challenge",
			key:  "valid-key",
			setup: func(store *ChallengeStore) {
				err := store.Set(ctx, "valid-key", []byte("valid-value"), time.Minute)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, value []byte) {
				assert.Equal(t, []byte("valid-value"), value)
			},
		},
		{
			name: "ok/empty_value",
			key:  "empty-key",
			setup: func(store *ChallengeStore) {
				err := store.Set(ctx, "empty-key", []byte{}, time.Minute)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, value []byte) {
				assert.Equal(t, []byte{}, value)
			},
		},
		{
			name:    "error/not_found",
			key:     "nonexistent-key",
			setup:   func(store *ChallengeStore) {},
			wantErr: true,
			errMsg:  "challenge not found",
		},
		{
			name: "error/expired_challenge",
			key:  "expired-key",
			setup: func(store *ChallengeStore) {
				err := store.Set(ctx, "expired-key", []byte("expired-value"), time.Millisecond)
				require.NoError(t, err)
				time.Sleep(2 * time.Millisecond) // Wait for expiry
			},
			wantErr: true,
			errMsg:  "challenge expired",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewChallengeStore()
			
			if tc.setup != nil {
				tc.setup(store)
			}
			
			value, err := store.Get(ctx, tc.key)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, value)
			} else {
				require.NoError(t, err)
				require.NotNil(t, value)
				
				if tc.validate != nil {
					tc.validate(t, value)
				}
			}
		})
	}
}

func TestChallengeStore_Delete(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		key     string
		setup   func(*ChallengeStore)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/delete_existing",
			key:  "delete-key",
			setup: func(store *ChallengeStore) {
				err := store.Set(ctx, "delete-key", []byte("delete-value"), time.Minute)
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name:    "ok/delete_nonexistent",
			key:     "nonexistent-key",
			setup:   func(store *ChallengeStore) {},
			wantErr: false, // Delete is idempotent
		},
		{
			name: "ok/delete_expired",
			key:  "expired-key",
			setup: func(store *ChallengeStore) {
				err := store.Set(ctx, "expired-key", []byte("expired-value"), time.Millisecond)
				require.NoError(t, err)
				time.Sleep(2 * time.Millisecond) // Wait for expiry
			},
			wantErr: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewChallengeStore()
			
			if tc.setup != nil {
				tc.setup(store)
			}
			
			err := store.Delete(ctx, tc.key)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify the challenge was deleted
				_, err = store.Get(ctx, tc.key)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "challenge not found")
			}
		})
	}
}

func TestChallengeStore_Take(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name     string
		key      string
		setup    func(*ChallengeStore)
		wantErr  bool
		errMsg   string
		validate func(*testing.T, []byte, *ChallengeStore)
	}{
		{
			name: "ok/take_valid_challenge",
			key:  "take-key",
			setup: func(store *ChallengeStore) {
				err := store.Set(ctx, "take-key", []byte("take-value"), time.Minute)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, value []byte, store *ChallengeStore) {
				assert.Equal(t, []byte("take-value"), value)
				
				// Verify challenge is gone
				_, err := store.Get(ctx, "take-key")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "challenge not found")
			},
		},
		{
			name: "ok/take_empty_value",
			key:  "empty-take-key",
			setup: func(store *ChallengeStore) {
				err := store.Set(ctx, "empty-take-key", []byte{}, time.Minute)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, value []byte, store *ChallengeStore) {
				assert.Equal(t, []byte{}, value)
				
				// Verify challenge is gone
				_, err := store.Get(ctx, "empty-take-key")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "challenge not found")
			},
		},
		{
			name:    "ok/take_nonexistent",
			key:     "nonexistent-key",
			setup:   func(store *ChallengeStore) {},
			wantErr: false,
			validate: func(t *testing.T, value []byte, store *ChallengeStore) {
				assert.Nil(t, value)
			},
		},
		{
			name: "ok/take_expired",
			key:  "expired-take-key",
			setup: func(store *ChallengeStore) {
				err := store.Set(ctx, "expired-take-key", []byte("expired-value"), time.Millisecond)
				require.NoError(t, err)
				time.Sleep(2 * time.Millisecond) // Wait for expiry
			},
			wantErr: false,
			validate: func(t *testing.T, value []byte, store *ChallengeStore) {
				assert.Nil(t, value)
				
				// Verify expired challenge was cleaned up
				_, err := store.Get(ctx, "expired-take-key")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "challenge not found")
			},
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewChallengeStore()
			
			if tc.setup != nil {
				tc.setup(store)
			}
			
			value, err := store.Take(ctx, tc.key)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				if tc.validate != nil {
					tc.validate(t, value, store)
				}
			}
		})
	}
}

func TestChallengeStore_TTLBehavior(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	store := NewChallengeStore()
	
	t.Run("challenge_expires_correctly", func(t *testing.T) {
		t.Parallel()
		
		ttl := 50 * time.Millisecond
		err := store.Set(ctx, "ttl-test", []byte("ttl-value"), ttl)
		require.NoError(t, err)
		
		// Should be available immediately
		value, err := store.Get(ctx, "ttl-test")
		require.NoError(t, err)
		assert.Equal(t, []byte("ttl-value"), value)
		
		// Should still be available before expiry
		time.Sleep(25 * time.Millisecond)
		value, err = store.Get(ctx, "ttl-test")
		require.NoError(t, err)
		assert.Equal(t, []byte("ttl-value"), value)
		
		// Should be expired after TTL
		time.Sleep(30 * time.Millisecond) // Total: 55ms > 50ms TTL
		_, err = store.Get(ctx, "ttl-test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "challenge expired")
	})
	
	t.Run("get_does_not_affect_expiry", func(t *testing.T) {
		t.Parallel()
		
		ttl := 100 * time.Millisecond
		err := store.Set(ctx, "no-renew", []byte("no-renew-value"), ttl)
		require.NoError(t, err)
		
		// Access multiple times
		for i := 0; i < 5; i++ {
			value, err := store.Get(ctx, "no-renew")
			require.NoError(t, err)
			assert.Equal(t, []byte("no-renew-value"), value)
			time.Sleep(10 * time.Millisecond)
		}
		
		// Should still expire at original time
		time.Sleep(60 * time.Millisecond) // Total: ~110ms > 100ms TTL
		_, err = store.Get(ctx, "no-renew")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "challenge expired")
	})
}

func TestChallengeStore_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	store := NewChallengeStore()
	
	t.Run("concurrent_sets", func(t *testing.T) {
		t.Parallel()
		
		const numGoroutines = 20
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		errors := make(chan error, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				key := "concurrent-" + string(rune('a'+idx))
				value := []byte("value-" + string(rune('a'+idx)))
				
				if err := store.Set(ctx, key, value, time.Minute); err != nil {
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
		
		// Verify all challenges were set
		for i := 0; i < numGoroutines; i++ {
			key := "concurrent-" + string(rune('a'+i))
			expectedValue := []byte("value-" + string(rune('a'+i)))
			
			value, err := store.Get(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, expectedValue, value)
		}
	})
	
	t.Run("concurrent_gets", func(t *testing.T) {
		t.Parallel()
		
		// Set up challenge
		err := store.Set(ctx, "concurrent-read", []byte("concurrent-value"), time.Minute)
		require.NoError(t, err)
		
		const numGoroutines = 30
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				value, err := store.Get(ctx, "concurrent-read")
				assert.NoError(t, err)
				assert.Equal(t, []byte("concurrent-value"), value)
			}()
		}
		
		wg.Wait()
	})
	
	t.Run("concurrent_takes", func(t *testing.T) {
		t.Parallel()
		
		// Set up multiple challenges
		const numChallenges = 10
		for i := 0; i < numChallenges; i++ {
			key := "take-" + string(rune('a'+i))
			value := []byte("take-value-" + string(rune('a'+i)))
			err := store.Set(ctx, key, value, time.Minute)
			require.NoError(t, err)
		}
		
		var wg sync.WaitGroup
		wg.Add(numChallenges)
		
		takenValues := make(chan []byte, numChallenges)
		
		for i := 0; i < numChallenges; i++ {
			go func(idx int) {
				defer wg.Done()
				
				key := "take-" + string(rune('a'+idx))
				value, err := store.Take(ctx, key)
				assert.NoError(t, err)
				if value != nil {
					takenValues <- value
				}
			}(i)
		}
		
		wg.Wait()
		close(takenValues)
		
		// All values should be taken exactly once
		var taken [][]byte
		for value := range takenValues {
			taken = append(taken, value)
		}
		
		assert.Len(t, taken, numChallenges)
		
		// Verify all challenges are gone
		for i := 0; i < numChallenges; i++ {
			key := "take-" + string(rune('a'+i))
			_, err := store.Get(ctx, key)
			assert.Error(t, err)
		}
	})
	
	t.Run("concurrent_set_delete", func(t *testing.T) {
		t.Parallel()
		
		const numOperations = 50
		var wg sync.WaitGroup
		wg.Add(numOperations * 2) // Set and delete operations
		
		for i := 0; i < numOperations; i++ {
			go func(idx int) {
				defer wg.Done()
				
				key := "setdel-" + string(rune('a'+idx%10))
				value := []byte("value-" + string(rune('a'+idx)))
				err := store.Set(ctx, key, value, time.Minute)
				assert.NoError(t, err)
			}(i)
			
			go func(idx int) {
				defer wg.Done()
				
				key := "setdel-" + string(rune('a'+idx%10))
				err := store.Delete(ctx, key)
				assert.NoError(t, err)
			}(i)
		}
		
		wg.Wait()
	})
}

func TestChallengeStore_DataIsolation(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("separate_stores_are_isolated", func(t *testing.T) {
		t.Parallel()
		
		store1 := NewChallengeStore()
		store2 := NewChallengeStore()
		
		// Set challenge in store1
		err := store1.Set(ctx, "isolated-key", []byte("isolated-value"), time.Minute)
		require.NoError(t, err)
		
		// Should not exist in store2
		_, err = store2.Get(ctx, "isolated-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "challenge not found")
		
		// Take from store2 should return nil
		value, err := store2.Take(ctx, "isolated-key")
		require.NoError(t, err)
		assert.Nil(t, value)
	})
	
	t.Run("modifications_dont_affect_returned_objects", func(t *testing.T) {
		t.Parallel()
		
		store := NewChallengeStore()
		
		// Set challenge
		originalValue := []byte("original-value")
		err := store.Set(ctx, "modify-test", originalValue, time.Minute)
		require.NoError(t, err)
		
		// Get challenge
		retrievedValue, err := store.Get(ctx, "modify-test")
		require.NoError(t, err)
		
		// Modify retrieved value
		for i := range retrievedValue {
			retrievedValue[i] = 'X'
		}
		
		// Get challenge again - should be unchanged
		unchangedValue, err := store.Get(ctx, "modify-test")
		require.NoError(t, err)
		
		assert.Equal(t, originalValue, unchangedValue)
		assert.NotEqual(t, retrievedValue, unchangedValue)
	})
	
	t.Run("set_value_modifications_dont_affect_store", func(t *testing.T) {
		t.Parallel()
		
		store := NewChallengeStore()
		
		// Create value and set it
		setValue := []byte("set-value")
		err := store.Set(ctx, "set-modify-test", setValue, time.Minute)
		require.NoError(t, err)
		
		// Modify original value after setting
		for i := range setValue {
			setValue[i] = 'Y'
		}
		
		// Retrieved value should be unchanged
		retrievedValue, err := store.Get(ctx, "set-modify-test")
		require.NoError(t, err)
		
		assert.Equal(t, []byte("set-value"), retrievedValue)
		assert.NotEqual(t, setValue, retrievedValue)
	})
}
package inmemory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKV(t *testing.T) {
	t.Parallel()
	
	kv := NewKV()
	assert.NotNil(t, kv)
}

func TestKV_Set(t *testing.T) {
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
			ttl:     0, // no expiration
			wantErr: false,
		},
		{
			name:    "ok/empty_value",
			key:     "empty-key",
			value:   []byte{},
			ttl:     0,
			wantErr: false,
		},
		{
			name:    "ok/nil_value",
			key:     "nil-key",
			value:   nil,
			ttl:     0,
			wantErr: false,
		},
		{
			name:    "ok/with_ttl",
			key:     "ttl-key",
			value:   []byte("ttl-value"),
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
			
			kv := NewKV()
			
			// For overwrite test, set an initial value
			if tc.name == "ok/overwrite_existing" {
				err := kv.Set(ctx, tc.key, []byte("old-value"), 0)
				require.NoError(t, err)
			}
			
			err := kv.Set(ctx, tc.key, tc.value, tc.ttl)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify the value was stored
				retrieved, err := kv.Get(ctx, tc.key)
				require.NoError(t, err)
				if tc.value == nil {
					// nil values become empty slices
					assert.Equal(t, []byte{}, retrieved)
				} else {
					assert.Equal(t, tc.value, retrieved)
				}
			}
		})
	}
}

func TestKV_Get(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name     string
		key      string
		setup    func(*KV)
		wantErr  bool
		validate func(*testing.T, []byte)
	}{
		{
			name: "ok/valid_key",
			key:  "valid-key",
			setup: func(kv *KV) {
				err := kv.Set(ctx, "valid-key", []byte("valid-value"), 0)
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
			setup: func(kv *KV) {
				err := kv.Set(ctx, "empty-key", []byte{}, 0)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, value []byte) {
				assert.Equal(t, []byte{}, value)
			},
		},
		{
			name: "ok/nil_value",
			key:  "nil-key",
			setup: func(kv *KV) {
				err := kv.Set(ctx, "nil-key", nil, 0)
				require.NoError(t, err)
			},
			wantErr: false,
			validate: func(t *testing.T, value []byte) {
				assert.Equal(t, []byte{}, value) // nil becomes empty slice
			},
		},
		{
			name:    "error/not_found",
			key:     "nonexistent-key",
			setup:   func(kv *KV) {},
			wantErr: true,
		},
		{
			name: "error/expired_key",
			key:  "expired-key",
			setup: func(kv *KV) {
				err := kv.Set(ctx, "expired-key", []byte("expired-value"), time.Millisecond)
				require.NoError(t, err)
				time.Sleep(2 * time.Millisecond) // Wait for expiry
			},
			wantErr: true,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			kv := NewKV()
			
			if tc.setup != nil {
				tc.setup(kv)
			}
			
			value, err := kv.Get(ctx, tc.key)
			
			if tc.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, store.ErrNotFound)
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

func TestKV_Del(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		key     string
		setup   func(*KV)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/delete_existing",
			key:  "delete-key",
			setup: func(kv *KV) {
				err := kv.Set(ctx, "delete-key", []byte("delete-value"), 0)
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name:    "ok/delete_nonexistent",
			key:     "nonexistent-key",
			setup:   func(kv *KV) {},
			wantErr: false, // Delete is idempotent
		},
		{
			name: "ok/delete_expired",
			key:  "expired-key",
			setup: func(kv *KV) {
				err := kv.Set(ctx, "expired-key", []byte("expired-value"), time.Millisecond)
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
			
			kv := NewKV()
			
			if tc.setup != nil {
				tc.setup(kv)
			}
			
			err := kv.Del(ctx, tc.key)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify the key was deleted
				_, err = kv.Get(ctx, tc.key)
				assert.Error(t, err)
				assert.ErrorIs(t, err, store.ErrNotFound)
			}
		})
	}
}

func TestKV_TTLBehavior(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	kv := NewKV()
	
	t.Run("key_expires_correctly", func(t *testing.T) {
		t.Parallel()
		
		ttl := 50 * time.Millisecond
		err := kv.Set(ctx, "ttl-test", []byte("ttl-value"), ttl)
		require.NoError(t, err)
		
		// Should be available immediately
		value, err := kv.Get(ctx, "ttl-test")
		require.NoError(t, err)
		assert.Equal(t, []byte("ttl-value"), value)
		
		// Should still be available before expiry
		time.Sleep(25 * time.Millisecond)
		value, err = kv.Get(ctx, "ttl-test")
		require.NoError(t, err)
		assert.Equal(t, []byte("ttl-value"), value)
		
		// Should be expired after TTL
		time.Sleep(30 * time.Millisecond) // Total: 55ms > 50ms TTL
		_, err = kv.Get(ctx, "ttl-test")
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrNotFound)
	})
	
	t.Run("zero_ttl_means_no_expiration", func(t *testing.T) {
		t.Parallel()
		
		err := kv.Set(ctx, "no-expiry", []byte("permanent-value"), 0)
		require.NoError(t, err)
		
		// Should be available after significant time
		time.Sleep(100 * time.Millisecond)
		value, err := kv.Get(ctx, "no-expiry")
		require.NoError(t, err)
		assert.Equal(t, []byte("permanent-value"), value)
	})
	
	t.Run("get_does_not_affect_expiry", func(t *testing.T) {
		t.Parallel()
		
		ttl := 100 * time.Millisecond
		err := kv.Set(ctx, "no-renew", []byte("no-renew-value"), ttl)
		require.NoError(t, err)
		
		// Access multiple times
		for i := 0; i < 5; i++ {
			value, err := kv.Get(ctx, "no-renew")
			require.NoError(t, err)
			assert.Equal(t, []byte("no-renew-value"), value)
			time.Sleep(10 * time.Millisecond)
		}
		
		// Should still expire at original time
		time.Sleep(60 * time.Millisecond) // Total: ~110ms > 100ms TTL
		_, err = kv.Get(ctx, "no-renew")
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrNotFound)
	})
	
	t.Run("overwrite_changes_expiry", func(t *testing.T) {
		t.Parallel()
		
		// Set with short TTL
		err := kv.Set(ctx, "overwrite-ttl", []byte("original"), 50*time.Millisecond)
		require.NoError(t, err)
		
		// Wait a bit
		time.Sleep(30 * time.Millisecond)
		
		// Overwrite with longer TTL
		err = kv.Set(ctx, "overwrite-ttl", []byte("updated"), 100*time.Millisecond)
		require.NoError(t, err)
		
		// Should be available after original TTL would have expired
		time.Sleep(30 * time.Millisecond) // Total: 60ms > original 50ms
		value, err := kv.Get(ctx, "overwrite-ttl")
		require.NoError(t, err)
		assert.Equal(t, []byte("updated"), value)
	})
}

func TestKV_Cleanup(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	kv := NewKV()
	
	t.Run("cleanup_removes_expired_keys", func(t *testing.T) {
		t.Parallel()
		
		// Set multiple keys with different expiration times
		err := kv.Set(ctx, "expired1", []byte("value1"), time.Millisecond)
		require.NoError(t, err)
		
		err = kv.Set(ctx, "expired2", []byte("value2"), time.Millisecond)
		require.NoError(t, err)
		
		err = kv.Set(ctx, "permanent", []byte("permanent-value"), 0)
		require.NoError(t, err)
		
		err = kv.Set(ctx, "long-lived", []byte("long-value"), time.Hour)
		require.NoError(t, err)
		
		// Wait for some to expire
		time.Sleep(2 * time.Millisecond)
		
		// Before cleanup, expired keys return ErrNotFound but still exist in map
		_, err = kv.Get(ctx, "expired1")
		assert.ErrorIs(t, err, store.ErrNotFound)
		
		// Run cleanup
		kv.Cleanup()
		
		// After cleanup, permanent and long-lived should still exist
		value, err := kv.Get(ctx, "permanent")
		require.NoError(t, err)
		assert.Equal(t, []byte("permanent-value"), value)
		
		value, err = kv.Get(ctx, "long-lived")
		require.NoError(t, err)
		assert.Equal(t, []byte("long-value"), value)
		
		// Expired keys should still return ErrNotFound
		_, err = kv.Get(ctx, "expired1")
		assert.ErrorIs(t, err, store.ErrNotFound)
		
		_, err = kv.Get(ctx, "expired2")
		assert.ErrorIs(t, err, store.ErrNotFound)
	})
}

func TestKV_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	kv := NewKV()
	
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
				
				if err := kv.Set(ctx, key, value, 0); err != nil {
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
		
		// Verify all keys were set
		for i := 0; i < numGoroutines; i++ {
			key := "concurrent-" + string(rune('a'+i))
			expectedValue := []byte("value-" + string(rune('a'+i)))
			
			value, err := kv.Get(ctx, key)
			assert.NoError(t, err)
			assert.Equal(t, expectedValue, value)
		}
	})
	
	t.Run("concurrent_gets", func(t *testing.T) {
		t.Parallel()
		
		// Set up key
		err := kv.Set(ctx, "concurrent-read", []byte("concurrent-value"), 0)
		require.NoError(t, err)
		
		const numGoroutines = 30
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				value, err := kv.Get(ctx, "concurrent-read")
				assert.NoError(t, err)
				assert.Equal(t, []byte("concurrent-value"), value)
			}()
		}
		
		wg.Wait()
	})
	
	t.Run("concurrent_set_get_del", func(t *testing.T) {
		t.Parallel()
		
		const numOperations = 50
		var wg sync.WaitGroup
		wg.Add(numOperations * 3) // Set, get, and delete operations
		
		for i := 0; i < numOperations; i++ {
			go func(idx int) {
				defer wg.Done()
				
				key := "setgetdel-" + string(rune('a'+idx%10))
				value := []byte("value-" + string(rune('a'+idx)))
				err := kv.Set(ctx, key, value, 0)
				assert.NoError(t, err)
			}(i)
			
			go func(idx int) {
				defer wg.Done()
				
				key := "setgetdel-" + string(rune('a'+idx%10))
				_, err := kv.Get(ctx, key)
				// May or may not exist depending on timing
				if err != nil {
					assert.ErrorIs(t, err, store.ErrNotFound)
				}
			}(i)
			
			go func(idx int) {
				defer wg.Done()
				
				key := "setgetdel-" + string(rune('a'+idx%10))
				err := kv.Del(ctx, key)
				assert.NoError(t, err)
			}(i)
		}
		
		wg.Wait()
	})
	
	t.Run("concurrent_ttl_operations", func(t *testing.T) {
		t.Parallel()
		
		const numGoroutines = 20
		var wg sync.WaitGroup
		wg.Add(numGoroutines * 2) // Set and get operations
		
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				key := "ttl-" + string(rune('a'+idx))
				value := []byte("ttl-value-" + string(rune('a'+idx)))
				err := kv.Set(ctx, key, value, 100*time.Millisecond)
				assert.NoError(t, err)
			}(i)
			
			go func(idx int) {
				defer wg.Done()
				
				// Try to get immediately and after delay
				key := "ttl-" + string(rune('a'+idx))
				
				// Should exist immediately
				_, err := kv.Get(ctx, key)
				if err != nil {
					assert.ErrorIs(t, err, store.ErrNotFound)
				}
				
				// May or may not exist after delay depending on timing
				time.Sleep(50 * time.Millisecond)
				_, err = kv.Get(ctx, key)
				if err != nil {
					assert.ErrorIs(t, err, store.ErrNotFound)
				}
			}(i)
		}
		
		wg.Wait()
	})
}

func TestKV_DataIsolation(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("separate_stores_are_isolated", func(t *testing.T) {
		t.Parallel()
		
		kv1 := NewKV()
		kv2 := NewKV()
		
		// Set key in kv1
		err := kv1.Set(ctx, "isolated-key", []byte("isolated-value"), 0)
		require.NoError(t, err)
		
		// Should not exist in kv2
		_, err = kv2.Get(ctx, "isolated-key")
		assert.Error(t, err)
		assert.ErrorIs(t, err, store.ErrNotFound)
	})
	
	t.Run("modifications_dont_affect_returned_objects", func(t *testing.T) {
		t.Parallel()
		
		kv := NewKV()
		
		// Set value
		originalValue := []byte("original-value")
		err := kv.Set(ctx, "modify-test", originalValue, 0)
		require.NoError(t, err)
		
		// Get value
		retrievedValue, err := kv.Get(ctx, "modify-test")
		require.NoError(t, err)
		
		// Modify retrieved value
		for i := range retrievedValue {
			retrievedValue[i] = 'X'
		}
		
		// Get value again - should be unchanged
		unchangedValue, err := kv.Get(ctx, "modify-test")
		require.NoError(t, err)
		
		assert.Equal(t, originalValue, unchangedValue)
		assert.NotEqual(t, retrievedValue, unchangedValue)
	})
	
	t.Run("set_value_modifications_dont_affect_store", func(t *testing.T) {
		t.Parallel()
		
		kv := NewKV()
		
		// Create value and set it
		setValue := []byte("set-value")
		err := kv.Set(ctx, "set-modify-test", setValue, 0)
		require.NoError(t, err)
		
		// Modify original value after setting
		for i := range setValue {
			setValue[i] = 'Y'
		}
		
		// Retrieved value should be unchanged
		retrievedValue, err := kv.Get(ctx, "set-modify-test")
		require.NoError(t, err)
		
		assert.Equal(t, []byte("set-value"), retrievedValue)
		assert.NotEqual(t, setValue, retrievedValue)
	})
}
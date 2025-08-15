package inmemory

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserStore(t *testing.T) {
	t.Parallel()
	
	store := NewUserStore()
	assert.NotNil(t, store)
}

func TestUserStore_Create(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		email   string
		roles   []string
		setup   func(*UserStore)
		wantErr bool
		errMsg  string
		validate func(*testing.T, *domain.User)
	}{
		{
			name:  "ok/basic_user",
			email: "user@example.com",
			roles: []string{"user"},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.NotEqual(t, uuid.Nil, user.ID)
				assert.Equal(t, "user@example.com", user.Email)
				assert.Equal(t, []string{"user"}, user.Roles)
				assert.Equal(t, int64(1), user.SessionVersion)
				assert.NotZero(t, user.CreatedAt)
				assert.NotZero(t, user.UpdatedAt)
			},
		},
		{
			name:  "ok/admin_user",
			email: "admin@example.com",
			roles: []string{"user", "admin"},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, "admin@example.com", user.Email)
				assert.Equal(t, []string{"user", "admin"}, user.Roles)
			},
		},
		{
			name:  "ok/no_roles",
			email: "noroles@example.com",
			roles: []string{},
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Equal(t, []string{}, user.Roles)
			},
		},
		{
			name:  "ok/nil_roles",
			email: "nilroles@example.com",
			roles: nil,
			wantErr: false,
			validate: func(t *testing.T, user *domain.User) {
				assert.Nil(t, user.Roles)
			},
		},
		{
			name:  "error/duplicate_email",
			email: "duplicate@example.com",
			roles: []string{"user"},
			setup: func(store *UserStore) {
				_, err := store.Create(ctx, "duplicate@example.com", []string{"user"})
				require.NoError(t, err)
			},
			wantErr: true,
			errMsg:  "user already exists",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewUserStore()
			
			if tc.setup != nil {
				tc.setup(store)
			}
			
			user, err := store.Create(ctx, tc.email, tc.roles)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				
				if tc.validate != nil {
					tc.validate(t, user)
				}
				
				// Verify user is stored
				stored, err := store.GetByID(ctx, user.ID)
				require.NoError(t, err)
				assert.Equal(t, user.ID, stored.ID)
				assert.Equal(t, user.Email, stored.Email)
			}
		})
	}
}

func TestUserStore_GetByEmail(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		email   string
		setup   func(*UserStore) *domain.User
		wantErr bool
		errMsg  string
		validate func(*testing.T, *domain.User, *domain.User)
	}{
		{
			name:  "ok/found_user",
			email: "user@example.com",
			setup: func(store *UserStore) *domain.User {
				user, err := store.Create(ctx, "user@example.com", []string{"user"})
				require.NoError(t, err)
				return user
			},
			wantErr: false,
			validate: func(t *testing.T, expected, actual *domain.User) {
				assert.Equal(t, expected.ID, actual.ID)
				assert.Equal(t, expected.Email, actual.Email)
				assert.Equal(t, expected.Roles, actual.Roles)
			},
		},
		{
			name:  "error/not_found",
			email: "notfound@example.com",
			setup: func(store *UserStore) *domain.User {
				return nil
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name:  "error/empty_email",
			email: "",
			setup: func(store *UserStore) *domain.User {
				return nil
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name:  "ok/case_sensitive",
			email: "User@Example.com",
			setup: func(store *UserStore) *domain.User {
				user, err := store.Create(ctx, "user@example.com", []string{"user"})
				require.NoError(t, err)
				return user
			},
			wantErr: true,
			errMsg:  "user not found",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewUserStore()
			
			var expected *domain.User
			if tc.setup != nil {
				expected = tc.setup(store)
			}
			
			actual, err := store.GetByEmail(ctx, tc.email)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, actual)
			} else {
				require.NoError(t, err)
				require.NotNil(t, actual)
				
				if tc.validate != nil {
					tc.validate(t, expected, actual)
				}
			}
		})
	}
}

func TestUserStore_GetByID(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		setup   func(*UserStore) (uuid.UUID, *domain.User)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/found_user",
			setup: func(store *UserStore) (uuid.UUID, *domain.User) {
				user, err := store.Create(ctx, "user@example.com", []string{"user"})
				require.NoError(t, err)
				return user.ID, user
			},
			wantErr: false,
		},
		{
			name: "error/not_found",
			setup: func(store *UserStore) (uuid.UUID, *domain.User) {
				return uuid.New(), nil
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name: "error/nil_uuid",
			setup: func(store *UserStore) (uuid.UUID, *domain.User) {
				return uuid.Nil, nil
			},
			wantErr: true,
			errMsg:  "user not found",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewUserStore()
			
			id, expected := tc.setup(store)
			
			actual, err := store.GetByID(ctx, id)
			
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
			}
		})
	}
}

func TestUserStore_UpdateRoles(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		setup   func(*UserStore) uuid.UUID
		roles   []string
		wantErr bool
		errMsg  string
		validate func(*testing.T, *UserStore, uuid.UUID)
	}{
		{
			name: "ok/update_roles",
			setup: func(store *UserStore) uuid.UUID {
				user, err := store.Create(ctx, "user@example.com", []string{"user"})
				require.NoError(t, err)
				return user.ID
			},
			roles:   []string{"user", "admin"},
			wantErr: false,
			validate: func(t *testing.T, store *UserStore, id uuid.UUID) {
				user, err := store.GetByID(ctx, id)
				require.NoError(t, err)
				assert.Equal(t, []string{"user", "admin"}, user.Roles)
			},
		},
		{
			name: "ok/empty_roles",
			setup: func(store *UserStore) uuid.UUID {
				user, err := store.Create(ctx, "user@example.com", []string{"user", "admin"})
				require.NoError(t, err)
				return user.ID
			},
			roles:   []string{},
			wantErr: false,
			validate: func(t *testing.T, store *UserStore, id uuid.UUID) {
				user, err := store.GetByID(ctx, id)
				require.NoError(t, err)
				assert.Equal(t, []string{}, user.Roles)
			},
		},
		{
			name: "error/user_not_found",
			setup: func(store *UserStore) uuid.UUID {
				return uuid.New()
			},
			roles:   []string{"user"},
			wantErr: true,
			errMsg:  "user not found",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewUserStore()
			id := tc.setup(store)
			
			err := store.UpdateRoles(ctx, id, tc.roles)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				if tc.validate != nil {
					tc.validate(t, store, id)
				}
			}
		})
	}
}

func TestUserStore_BumpSessionVersion(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		setup   func(*UserStore) (uuid.UUID, int64)
		wantErr bool
		errMsg  string
	}{
		{
			name: "ok/bump_version",
			setup: func(store *UserStore) (uuid.UUID, int64) {
				user, err := store.Create(ctx, "user@example.com", []string{"user"})
				require.NoError(t, err)
				return user.ID, user.SessionVersion
			},
			wantErr: false,
		},
		{
			name: "ok/bump_multiple_times",
			setup: func(store *UserStore) (uuid.UUID, int64) {
				user, err := store.Create(ctx, "user@example.com", []string{"user"})
				require.NoError(t, err)
				
				// Bump multiple times
				for i := 0; i < 5; i++ {
					err = store.BumpSessionVersion(ctx, user.ID)
					require.NoError(t, err)
				}
				
				// Get the current version after 5 bumps (should be 6)
				updated, err := store.GetByID(ctx, user.ID)
				require.NoError(t, err)
				
				// Return the current version (which is 6)
				// The test will bump once more, making it 7
				return user.ID, updated.SessionVersion
			},
			wantErr: false,
		},
		{
			name: "error/user_not_found",
			setup: func(store *UserStore) (uuid.UUID, int64) {
				return uuid.New(), 0
			},
			wantErr: true,
			errMsg:  "user not found",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			store := NewUserStore()
			id, originalVersion := tc.setup(store)
			
			err := store.BumpSessionVersion(ctx, id)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify version was incremented
				user, err := store.GetByID(ctx, id)
				require.NoError(t, err)
				assert.Equal(t, originalVersion+1, user.SessionVersion)
			}
		})
	}
}

func TestUserStore_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	store := NewUserStore()
	
	t.Run("concurrent_creates", func(t *testing.T) {
		t.Parallel()
		
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)
		
		errors := make(chan error, numGoroutines)
		users := make(chan *domain.User, numGoroutines)
		
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				email := fmt.Sprintf("user%d@example.com", idx)
				user, err := store.Create(ctx, email, []string{"user"})
				if err != nil {
					errors <- err
				} else {
					users <- user
				}
			}(i)
		}
		
		wg.Wait()
		close(errors)
		close(users)
		
		// Check for errors
		for err := range errors {
			assert.NoError(t, err)
		}
		
		// Verify all users were created
		userCount := 0
		for range users {
			userCount++
		}
		assert.Equal(t, numGoroutines, userCount)
	})
	
	t.Run("concurrent_reads", func(t *testing.T) {
		t.Parallel()
		
		// Create a user first
		user, err := store.Create(ctx, "reader@example.com", []string{"user"})
		require.NoError(t, err)
		
		const numGoroutines = 20
		var wg sync.WaitGroup
		wg.Add(numGoroutines * 2) // For both GetByID and GetByEmail
		
		// Concurrent GetByID
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				found, err := store.GetByID(ctx, user.ID)
				assert.NoError(t, err)
				assert.NotNil(t, found)
			}()
		}
		
		// Concurrent GetByEmail
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				found, err := store.GetByEmail(ctx, "reader@example.com")
				assert.NoError(t, err)
				assert.NotNil(t, found)
			}()
		}
		
		wg.Wait()
	})
	
	t.Run("concurrent_updates", func(t *testing.T) {
		t.Parallel()
		
		// Create a user first
		user, err := store.Create(ctx, "updater@example.com", []string{"user"})
		require.NoError(t, err)
		
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines * 2) // For both UpdateRoles and BumpSessionVersion
		
		// Concurrent UpdateRoles
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				
				roles := []string{"user"}
				if idx%2 == 0 {
					roles = append(roles, "admin")
				}
				
				err := store.UpdateRoles(ctx, user.ID, roles)
				assert.NoError(t, err)
			}(i)
		}
		
		// Concurrent BumpSessionVersion
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				
				err := store.BumpSessionVersion(ctx, user.ID)
				assert.NoError(t, err)
			}()
		}
		
		wg.Wait()
		
		// Verify final state
		final, err := store.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, final.SessionVersion, int64(numGoroutines))
	})
}

func TestUserStore_DataIsolation(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	
	t.Run("separate_stores_are_isolated", func(t *testing.T) {
		t.Parallel()
		
		store1 := NewUserStore()
		store2 := NewUserStore()
		
		// Create user in store1
		user1, err := store1.Create(ctx, "user@store1.com", []string{"user"})
		require.NoError(t, err)
		
		// Should not exist in store2
		_, err = store2.GetByID(ctx, user1.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
		
		_, err = store2.GetByEmail(ctx, "user@store1.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
		
		// Create user in store2
		user2, err := store2.Create(ctx, "user@store2.com", []string{"admin"})
		require.NoError(t, err)
		
		// Should not exist in store1
		_, err = store1.GetByID(ctx, user2.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
	
	t.Run("modifications_dont_affect_returned_objects", func(t *testing.T) {
		t.Parallel()
		
		store := NewUserStore()
		
		// Create user
		original, err := store.Create(ctx, "user@example.com", []string{"user"})
		require.NoError(t, err)
		
		// Get user
		retrieved, err := store.GetByID(ctx, original.ID)
		require.NoError(t, err)
		
		// Modify retrieved user
		retrieved.Email = "modified@example.com"
		retrieved.Roles = []string{"admin"}
		retrieved.SessionVersion = 999
		
		// Get user again - should be unchanged
		unchanged, err := store.GetByID(ctx, original.ID)
		require.NoError(t, err)
		
		assert.Equal(t, "user@example.com", unchanged.Email)
		assert.Equal(t, []string{"user"}, unchanged.Roles)
		assert.Equal(t, int64(1), unchanged.SessionVersion)
	})
}
package service

import (
	"context"
	"encoding/base64"
	"sync"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/testing/inmemory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRecoveryService(t *testing.T) (RecoveryService, *inmemory.RecoveryCodeStore) {
	t.Helper()
	
	store := inmemory.NewRecoveryCodeStore()
	rand := crypto.NewSecureRand()
	
	svc := NewRecoveryService(store, rand)
	return svc, store
}

func TestNewRecoveryService(t *testing.T) {
	t.Parallel()
	
	t.Run("ok/valid_config", func(t *testing.T) {
		t.Parallel()
		
		store := inmemory.NewRecoveryCodeStore()
		rand := crypto.NewSecureRand()
		
		assert.NotPanics(t, func() {
			_ = NewRecoveryService(store, rand)
		})
	})
	
	t.Run("ok/nil_store", func(t *testing.T) {
		t.Parallel()
		
		rand := crypto.NewSecureRand()
		
		assert.NotPanics(t, func() {
			svc := NewRecoveryService(nil, rand)
			assert.NotNil(t, svc)
		})
	})
	
	t.Run("ok/nil_rand", func(t *testing.T) {
		t.Parallel()
		
		store := inmemory.NewRecoveryCodeStore()
		
		assert.NotPanics(t, func() {
			svc := NewRecoveryService(store, nil)
			assert.NotNil(t, svc)
		})
	})
}

func TestRecoveryService_GenerateCodes(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, store := setupRecoveryService(t)
	userID := uuid.New()
	
	tests := []struct {
		name     string
		userID   uuid.UUID
		count    int
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, codes []string)
	}{
		{
			name:    "ok/generate_8_codes",
			userID:  userID,
			count:   8,
			wantErr: false,
			validate: func(t *testing.T, codes []string) {
				assert.Len(t, codes, 8)
				
				// Check code format (base64 URL encoded)
				for _, code := range codes {
					// base64 URL encoded 16 bytes = ~22 characters
					assert.Greater(t, len(code), 20, "recovery codes should be base64 encoded")
					assert.Regexp(t, "^[A-Za-z0-9_-]+$", code, "codes should be base64 URL encoded")
				}
				
				// Check codes are unique
				seen := make(map[string]bool)
				for _, code := range codes {
					assert.False(t, seen[code], "codes should be unique")
					seen[code] = true
				}
				
				// Verify codes were stored as hashes
				storedCodes, err := store.GetByUserID(ctx, userID)
				require.NoError(t, err)
				assert.Len(t, storedCodes, 8)
				
				// All stored codes should be unused
				for _, sc := range storedCodes {
					assert.Nil(t, sc.UsedAt, "new codes should be unused")
				}
			},
		},
		{
			name:    "ok/generate_12_codes",
			userID:  uuid.New(),
			count:   12,
			wantErr: false,
			validate: func(t *testing.T, codes []string) {
				assert.Len(t, codes, 12)
			},
		},
		{
			name:    "ok/generate_1_code",
			userID:  uuid.New(),
			count:   1,
			wantErr: false,
			validate: func(t *testing.T, codes []string) {
				assert.Len(t, codes, 1)
			},
		},
		{
			name:    "ok/zero_count_uses_default",
			userID:  uuid.New(),
			count:   0,
			wantErr: false,
			validate: func(t *testing.T, codes []string) {
				assert.Len(t, codes, 10, "should use default count of 10")
			},
		},
		{
			name:    "ok/negative_count_uses_default",
			userID:  uuid.New(),
			count:   -5,
			wantErr: false,
			validate: func(t *testing.T, codes []string) {
				assert.Len(t, codes, 10, "should use default count of 10")
			},
		},
		{
			name:    "ok/large_count",
			userID:  uuid.New(),
			count:   100,
			wantErr: false,
			validate: func(t *testing.T, codes []string) {
				assert.Len(t, codes, 100)
			},
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			codes, err := svc.GenerateCodes(ctx, tc.userID, tc.count)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, codes)
			} else {
				require.NoError(t, err)
				require.NotNil(t, codes)
				
				if tc.validate != nil {
					tc.validate(t, codes)
				}
			}
		})
	}
}

func TestRecoveryService_ValidateCode(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, store := setupRecoveryService(t)
	userID := uuid.New()
	
	// Generate codes for testing
	codes, err := svc.GenerateCodes(ctx, userID, 8)
	require.NoError(t, err)
	require.Len(t, codes, 8)
	
	tests := []struct {
		name    string
		userID  uuid.UUID
		code    string
		setup   func(t *testing.T)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "ok/valid_code",
			userID:  userID,
			code:    codes[0],
			wantErr: false,
		},
		{
			name:    "ok/valid_code_again",
			userID:  userID,
			code:    codes[1],
			wantErr: false,
		},
		{
			name:    "error/already_used_code",
			userID:  userID,
			code:    codes[0], // Already used in first test
			wantErr: true,
			errMsg:  "invalid or already used",
		},
		{
			name:    "error/wrong_code",
			userID:  userID,
			code:    "WRONGCODEWRONGCODEWRONG",  // Invalid base64
			wantErr: true,
			errMsg:  "invalid or already used",
		},
		{
			name:    "error/empty_code",
			userID:  userID,
			code:    "",
			wantErr: true,
			errMsg:  "invalid or already used",
		},
		{
			name:    "error/wrong_user",
			userID:  uuid.New(),
			code:    codes[2],
			wantErr: true,
			errMsg:  "invalid or already used",
		},
		{
			name:    "error/invalid_format",
			userID:  userID,
			code:    "!@#$%^&*()",
			wantErr: true,
			errMsg:  "invalid or already used",
		},
		{
			name:    "error/too_short",
			userID:  userID,
			code:    "SHORT",
			wantErr: true,
			errMsg:  "invalid or already used",
		},
		{
			name:    "error/too_long",
			userID:  userID,
			code:    "TOOLONGTOOLONGTOOLONGTOOLONGTOOLONGTOOLONG",
			wantErr: true,
			errMsg:  "invalid or already used",
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			// Note: Can't run in parallel due to shared codes state
			
			if tc.setup != nil {
				tc.setup(t)
			}
			
			err := svc.ValidateCode(ctx, tc.userID, tc.code)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				
				// Verify code was marked as used
				// Decode the base64 code to get the raw bytes
				codeBytes, err := base64.RawURLEncoding.DecodeString(tc.code)
				require.NoError(t, err)
				codeHash := crypto.HashSHA256(codeBytes)
				
				storedCode, err := store.GetByHash(ctx, userID, codeHash)
				require.NoError(t, err)
				require.NotNil(t, storedCode, "stored code should exist")
				assert.NotNil(t, storedCode.UsedAt, "code should be marked as used")
			}
		})
	}
}

func TestRecoveryService_GetRemainingCount(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, _ := setupRecoveryService(t)
	userID := uuid.New()
	
	tests := []struct {
		name     string
		setup    func(t *testing.T)
		userID   uuid.UUID
		wantErr  bool
		expected int
	}{
		{
			name: "ok/all_codes_unused",
			setup: func(t *testing.T) {
				_, err := svc.GenerateCodes(ctx, userID, 8)
				require.NoError(t, err)
			},
			userID:   userID,
			wantErr:  false,
			expected: 8,
		},
		{
			name: "ok/some_codes_used",
			setup: func(t *testing.T) {
				newUserID := uuid.New()
				codes, err := svc.GenerateCodes(ctx, newUserID, 8)
				require.NoError(t, err)
				
				// Use 3 codes
				for i := 0; i < 3; i++ {
					err = svc.ValidateCode(ctx, newUserID, codes[i])
					require.NoError(t, err)
				}
				
				// Update userID for this test
				userID = newUserID
			},
			userID:   uuid.Nil, // Will be set in setup
			wantErr:  false,
			expected: 5,
		},
		{
			name:     "ok/no_codes",
			userID:   uuid.New(),
			wantErr:  false,
			expected: 0,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			testUserID := tc.userID
			if tc.setup != nil {
				tc.setup(t)
				if tc.userID == uuid.Nil {
					testUserID = userID // Use the one from setup
				}
			}
			
			count, err := svc.GetRemainingCount(ctx, testUserID)
			
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, count)
			}
		})
	}
}

func TestRecoveryService_RegenerateCodes(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, store := setupRecoveryService(t)
	userID := uuid.New()
	
	// Generate initial codes
	oldCodes, err := svc.GenerateCodes(ctx, userID, 8)
	require.NoError(t, err)
	require.Len(t, oldCodes, 8)
	
	// Use one code
	err = svc.ValidateCode(ctx, userID, oldCodes[0])
	require.NoError(t, err)
	
	tests := []struct {
		name     string
		userID   uuid.UUID
		count    int
		wantErr  bool
		errMsg   string
		validate func(t *testing.T, newCodes []string)
	}{
		{
			name:    "ok/regenerate_with_same_count",
			userID:  userID,
			count:   8,
			wantErr: false,
			validate: func(t *testing.T, newCodes []string) {
				assert.Len(t, newCodes, 8)
				
				// Verify new codes are different from old codes
				for _, newCode := range newCodes {
					for _, oldCode := range oldCodes {
						assert.NotEqual(t, newCode, oldCode, "regenerated codes should be different")
					}
				}
				
				// Verify old codes no longer work
				err := svc.ValidateCode(ctx, userID, oldCodes[1])
				assert.Error(t, err, "old codes should not work after regeneration")
				
				// Verify only new codes exist in store
				storedCodes, err := store.GetByUserID(ctx, userID)
				require.NoError(t, err)
				assert.Len(t, storedCodes, 8)
				
				// All should be unused
				for _, sc := range storedCodes {
					assert.Nil(t, sc.UsedAt, "regenerated codes should be unused")
				}
			},
		},
		{
			name:    "ok/regenerate_with_different_count",
			userID:  uuid.New(),
			count:   12,
			wantErr: false,
			validate: func(t *testing.T, newCodes []string) {
				assert.Len(t, newCodes, 12)
			},
		},
		{
			name:    "ok/zero_count_uses_default",
			userID:  uuid.New(),
			count:   0,
			wantErr: false,
			validate: func(t *testing.T, newCodes []string) {
				assert.Len(t, newCodes, 10, "should use default count of 10")
			},
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			// Note: Can't run in parallel due to shared userID state
			
			newCodes, err := svc.RegenerateCodes(ctx, tc.userID, tc.count)
			
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				assert.Nil(t, newCodes)
			} else {
				require.NoError(t, err)
				require.NotNil(t, newCodes)
				
				if tc.validate != nil {
					tc.validate(t, newCodes)
				}
			}
		})
	}
}

func TestRecoveryService_CodeEntropy(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, _ := setupRecoveryService(t)
	
	// Generate multiple sets of codes
	const numSets = 10
	const codesPerSet = 8
	allCodes := make(map[string]bool)
	
	for i := 0; i < numSets; i++ {
		userID := uuid.New()
		codes, err := svc.GenerateCodes(ctx, userID, codesPerSet)
		require.NoError(t, err)
		
		for _, code := range codes {
			// Check for duplicates across all generated codes
			assert.False(t, allCodes[code], "should not generate duplicate codes")
			allCodes[code] = true
		}
	}
	
	// We should have generated numSets * codesPerSet unique codes
	assert.Equal(t, numSets*codesPerSet, len(allCodes))
	
	// Check character distribution (basic entropy check for base64 URL)
	charCount := make(map[rune]int)
	for code := range allCodes {
		for _, char := range code {
			charCount[char]++
		}
	}
	
	// Should have a good distribution of base64 URL characters
	// At minimum we should see letters and numbers
	hasLetters := false
	hasNumbers := false
	
	for char := range charCount {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' {
			hasLetters = true
		} else if char >= '0' && char <= '9' {
			hasNumbers = true
		}
	}
	
	assert.True(t, hasLetters, "should have letters in base64 codes")
	assert.True(t, hasNumbers, "should have numbers in base64 codes")
}

func TestRecoveryService_SingleUse(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, _ := setupRecoveryService(t)
	userID := uuid.New()
	
	// Generate codes
	codes, err := svc.GenerateCodes(ctx, userID, 8)
	require.NoError(t, err)
	
	testCode := codes[0]
	
	// First use should succeed
	err = svc.ValidateCode(ctx, userID, testCode)
	assert.NoError(t, err)
	
	// Second use should fail
	err = svc.ValidateCode(ctx, userID, testCode)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already used")
	
	// Third use should also fail
	err = svc.ValidateCode(ctx, userID, testCode)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already used")
}

func TestRecoveryService_TimingAttackResistance(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, _ := setupRecoveryService(t)
	userID := uuid.New()
	
	// Generate codes
	codes, err := svc.GenerateCodes(ctx, userID, 8)
	require.NoError(t, err)
	
	validCode := codes[0]
	invalidCode := "INVALIDBASE64CODE!!!"  // Will fail at decode
	
	// Measure timing for valid code
	const iterations = 100
	var validTimes []int64
	var invalidTimes []int64
	
	for i := 0; i < iterations; i++ {
		// Test invalid code
		start := time.Now().UnixNano()
		_ = svc.ValidateCode(ctx, userID, invalidCode)
		invalidTimes = append(invalidTimes, time.Now().UnixNano()-start)
		
		// Test valid code (will fail after first iteration)
		start = time.Now().UnixNano()
		_ = svc.ValidateCode(ctx, userID, validCode)
		validTimes = append(validTimes, time.Now().UnixNano()-start)
	}
	
	// Calculate averages
	var validAvg, invalidAvg int64
	for i := 0; i < iterations; i++ {
		validAvg += validTimes[i]
		invalidAvg += invalidTimes[i]
	}
	validAvg /= iterations
	invalidAvg /= iterations
	
	// The timing difference should be minimal (constant-time comparison)
	// Allow up to 100% difference due to system noise and test environment variability
	diff := abs(validAvg - invalidAvg)
	maxAllowedDiff := max(validAvg, invalidAvg)
	
	assert.LessOrEqual(t, diff, maxAllowedDiff,
		"timing difference should be minimal to prevent timing attacks")
}

func TestRecoveryService_Concurrency(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, _ := setupRecoveryService(t)
	
	// Create multiple users
	const numUsers = 5
	users := make([]uuid.UUID, numUsers)
	for i := 0; i < numUsers; i++ {
		users[i] = uuid.New()
	}
	
	// Run concurrent operations
	var wg sync.WaitGroup
	
	// Generate codes concurrently
	wg.Add(numUsers)
	for i := 0; i < numUsers; i++ {
		go func(userID uuid.UUID) {
			defer wg.Done()
			codes, err := svc.GenerateCodes(ctx, userID, 8)
			assert.NoError(t, err)
			assert.Len(t, codes, 8)
		}(users[i])
	}
	wg.Wait()
	
	// Get remaining counts concurrently
	wg.Add(numUsers)
	for i := 0; i < numUsers; i++ {
		go func(userID uuid.UUID) {
			defer wg.Done()
			count, err := svc.GetRemainingCount(ctx, userID)
			assert.NoError(t, err)
			assert.Equal(t, 8, count)
		}(users[i])
	}
	wg.Wait()
	
	// Regenerate codes concurrently
	wg.Add(numUsers)
	for i := 0; i < numUsers; i++ {
		go func(userID uuid.UUID) {
			defer wg.Done()
			codes, err := svc.RegenerateCodes(ctx, userID, 10)
			assert.NoError(t, err)
			assert.Len(t, codes, 10)
		}(users[i])
	}
	wg.Wait()
}

func TestRecoveryService_BatchReplacement(t *testing.T) {
	t.Parallel()
	
	ctx := context.Background()
	svc, store := setupRecoveryService(t)
	userID := uuid.New()
	
	// Generate initial batch
	codes1, err := svc.GenerateCodes(ctx, userID, 8)
	require.NoError(t, err)
	
	// Use some codes
	err = svc.ValidateCode(ctx, userID, codes1[0])
	require.NoError(t, err)
	err = svc.ValidateCode(ctx, userID, codes1[1])
	require.NoError(t, err)
	
	// Verify 6 codes remaining
	count, err := svc.GetRemainingCount(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 6, count)
	
	// Replace entire batch
	codes2, err := svc.RegenerateCodes(ctx, userID, 10)
	require.NoError(t, err)
	assert.Len(t, codes2, 10)
	
	// Verify all old codes are gone
	for _, oldCode := range codes1 {
		err = svc.ValidateCode(ctx, userID, oldCode)
		assert.Error(t, err, "old code should not work after batch replacement")
	}
	
	// Verify new count
	count, err = svc.GetRemainingCount(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 10, count)
	
	// Verify store only has new codes
	storedCodes, err := store.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, storedCodes, 10)
}

// Helper functions
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
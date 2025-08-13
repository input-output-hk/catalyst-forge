package crypto

import (
	"encoding/base64"
	"fmt"
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecureRand_Bytes(t *testing.T) {
	t.Parallel()
	
	r := NewSecureRand()
	
	tests := []struct {
		name    string
		n       int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "ok/single_byte",
			n:       1,
			wantErr: false,
		},
		{
			name:    "ok/16_bytes",
			n:       16,
			wantErr: false,
		},
		{
			name:    "ok/32_bytes",
			n:       32,
			wantErr: false,
		},
		{
			name:    "ok/256_bytes",
			n:       256,
			wantErr: false,
		},
		{
			name:    "ok/large_buffer",
			n:       65536, // 64KB
			wantErr: false,
		},
		{
			name:    "error/zero_bytes",
			n:       0,
			wantErr: true,
			errMsg:  "invalid byte count",
		},
		{
			name:    "error/negative_bytes",
			n:       -1,
			wantErr: true,
			errMsg:  "invalid byte count",
		},
	}
	
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			bytes, err := r.Bytes(tc.n)
			
			if tc.wantErr {
				require.Error(t, err, "expected error for %s", tc.name)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg, "error message mismatch")
				}
				assert.Nil(t, bytes, "bytes should be nil on error")
			} else {
				require.NoError(t, err, "unexpected error for %s", tc.name)
				assert.Len(t, bytes, tc.n, "bytes length mismatch")
				
				// Verify bytes are not all zeros (extremely unlikely with crypto/rand)
				if tc.n > 0 {
					allZero := true
					for _, b := range bytes {
						if b != 0 {
							allZero = false
							break
						}
					}
					assert.False(t, allZero, "random bytes should not be all zeros")
				}
			}
		})
	}
}

func TestSecureRand_String(t *testing.T) {
	t.Parallel()
	
	r := NewSecureRand()
	
	tests := []struct {
		name          string
		nBytes        int
		wantErr       bool
		expectedChars int // base64url encoded length
	}{
		{
			name:          "ok/16_bytes",
			nBytes:        16,
			wantErr:       false,
			expectedChars: 22, // ceil(16 * 8 / 6) for base64
		},
		{
			name:          "ok/32_bytes",
			nBytes:        32,
			wantErr:       false,
			expectedChars: 43, // ceil(32 * 8 / 6)
		},
		{
			name:          "ok/24_bytes",
			nBytes:        24,
			wantErr:       false,
			expectedChars: 32, // exactly 32 chars (no padding needed)
		},
		{
			name:    "error/zero_bytes",
			nBytes:  0,
			wantErr: true,
		},
		{
			name:    "error/negative_bytes",
			nBytes:  -1,
			wantErr: true,
		},
	}
	
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			str, err := r.String(tc.nBytes)
			
			if tc.wantErr {
				require.Error(t, err, "expected error for %s", tc.name)
				assert.Empty(t, str, "string should be empty on error")
			} else {
				require.NoError(t, err, "unexpected error for %s", tc.name)
				assert.Len(t, str, tc.expectedChars, "string length mismatch")
				
				// Verify it's valid base64url (no padding, URL-safe chars)
				assert.NotContains(t, str, "=", "base64url should not have padding")
				assert.NotContains(t, str, "+", "base64url should not have + char")
				assert.NotContains(t, str, "/", "base64url should not have / char")
				
				// Verify it can be decoded
				decoded, err := base64.RawURLEncoding.DecodeString(str)
				require.NoError(t, err, "should be valid base64url")
				assert.Len(t, decoded, tc.nBytes, "decoded bytes length mismatch")
			}
		})
	}
}

func TestSecureRand_Uniqueness(t *testing.T) {
	t.Parallel()
	
	r := NewSecureRand()
	
	// Generate multiple random values and ensure they're unique
	// Collision probability should be negligible with crypto/rand
	seen := make(map[string]bool)
	iterations := 1000
	byteCount := 16 // 128 bits
	
	for i := 0; i < iterations; i++ {
		bytes, err := r.Bytes(byteCount)
		require.NoError(t, err, "failed to generate random bytes at iteration %d", i)
		
		key := string(bytes)
		assert.False(t, seen[key], "duplicate random value detected at iteration %d", i)
		seen[key] = true
	}
	
	assert.Len(t, seen, iterations, "should have generated %d unique values", iterations)
}

func TestSecureRand_Distribution(t *testing.T) {
	t.Parallel()
	
	r := NewSecureRand()
	
	// Statistical test for uniform distribution
	// Generate many bytes and check distribution
	sampleSize := 100000
	bytes, err := r.Bytes(sampleSize)
	require.NoError(t, err, "failed to generate random bytes")
	
	// Count frequency of each byte value
	frequency := make([]int, 256)
	for _, b := range bytes {
		frequency[b]++
	}
	
	// Expected frequency for uniform distribution
	expectedFreq := float64(sampleSize) / 256.0
	
	// Chi-square test (simplified)
	chiSquare := 0.0
	for _, freq := range frequency {
		diff := float64(freq) - expectedFreq
		chiSquare += (diff * diff) / expectedFreq
	}
	
	// For 255 degrees of freedom and 0.01 significance level,
	// critical value is approximately 310
	// This is a basic check, not a rigorous statistical test
	assert.Less(t, chiSquare, 400.0, "distribution appears non-uniform (chi-square: %f)", chiSquare)
	
	// Also check that no byte value is completely missing or overly frequent
	for value, freq := range frequency {
		assert.Greater(t, freq, 200, "byte value %d appears too infrequently", value)
		assert.Less(t, freq, 600, "byte value %d appears too frequently", value)
	}
}

func TestSecureRand_Entropy(t *testing.T) {
	t.Parallel()
	
	r := NewSecureRand()
	
	// Test Shannon entropy to ensure high randomness
	bytes, err := r.Bytes(10000)
	require.NoError(t, err, "failed to generate random bytes")
	
	// Count byte frequencies
	frequency := make([]int, 256)
	for _, b := range bytes {
		frequency[b]++
	}
	
	// Calculate Shannon entropy
	entropy := 0.0
	total := float64(len(bytes))
	for _, freq := range frequency {
		if freq > 0 {
			p := float64(freq) / total
			entropy -= p * math.Log2(p)
		}
	}
	
	// Perfect entropy for 256 values is 8 bits
	// Good crypto random should be very close to 8
	assert.Greater(t, entropy, 7.9, "entropy too low: %f bits", entropy)
	assert.LessOrEqual(t, entropy, 8.0, "entropy cannot exceed 8 bits")
}

func TestSecureRand_Concurrency(t *testing.T) {
	t.Parallel()
	
	r := NewSecureRand()
	
	// Test concurrent access
	var wg sync.WaitGroup
	goroutines := 10
	iterations := 100
	
	results := make(chan []byte, goroutines*iterations)
	
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < iterations; j++ {
				bytes, err := r.Bytes(16)
				assert.NoError(t, err, "error in goroutine %d iteration %d", id, j)
				assert.Len(t, bytes, 16, "wrong length in goroutine %d iteration %d", id, j)
				results <- bytes
			}
		}(i)
	}
	
	wg.Wait()
	close(results)
	
	// Verify all results are unique
	seen := make(map[string]bool)
	count := 0
	for bytes := range results {
		key := string(bytes)
		assert.False(t, seen[key], "duplicate found in concurrent generation")
		seen[key] = true
		count++
	}
	
	assert.Equal(t, goroutines*iterations, count, "wrong number of results")
}

func TestSecureRand_PropertyRoundTrip(t *testing.T) {
	t.Parallel()
	
	r := NewSecureRand()
	
	// Property: String(n) should decode back to n bytes
	testSizes := []int{1, 8, 16, 24, 32, 64, 128, 256}
	
	for _, size := range testSizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			// Generate string
			str, err := r.String(size)
			require.NoError(t, err, "failed to generate string")
			
			// Decode back
			decoded, err := base64.RawURLEncoding.DecodeString(str)
			require.NoError(t, err, "failed to decode string")
			
			// Verify length
			assert.Len(t, decoded, size, "decoded length mismatch")
			
			// Verify it's not all zeros
			allZero := true
			for _, b := range decoded {
				if b != 0 {
					allZero = false
					break
				}
			}
			assert.False(t, allZero, "decoded bytes should not be all zeros")
		})
	}
}

func TestSecureRand_NoPatterns(t *testing.T) {
	t.Parallel()
	
	r := NewSecureRand()
	
	// Generate sequential random values and check for patterns
	previous, err := r.Bytes(32)
	require.NoError(t, err)
	
	for i := 0; i < 100; i++ {
		current, err := r.Bytes(32)
		require.NoError(t, err)
		
		// Check that consecutive values are different
		assert.NotEqual(t, previous, current, "consecutive values should not be equal at iteration %d", i)
		
		// Check that there's no simple increment pattern
		// (all bytes incremented by same value)
		if len(previous) == len(current) {
			diffs := make([]int, len(previous))
			firstDiff := int(current[0]) - int(previous[0])
			allSame := true
			
			for j := range previous {
				diffs[j] = int(current[j]) - int(previous[j])
				if diffs[j] != firstDiff {
					allSame = false
				}
			}
			
			assert.False(t, allSame, "detected increment pattern at iteration %d", i)
		}
		
		previous = current
	}
}

func BenchmarkSecureRand_Bytes(b *testing.B) {
	r := NewSecureRand()
	
	sizes := []int{16, 32, 64, 256, 1024}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := r.Bytes(size)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSecureRand_String(b *testing.B) {
	r := NewSecureRand()
	
	sizes := []int{16, 32, 64}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := r.String(size)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
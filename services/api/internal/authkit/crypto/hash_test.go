package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashSHA256(t *testing.T) {
	t.Parallel()

	// Test vectors from https://www.di-mgt.com.au/sha_testvectors.html
	tests := []struct {
		name     string
		input    []byte
		expected string // hex encoded
	}{
		{
			name:     "empty",
			input:    []byte{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "abc",
			input:    []byte("abc"),
			expected: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		},
		{
			name:     "message_digest",
			input:    []byte("abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq"),
			expected: "248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1",
		},
		{
			name:     "alphabet",
			input:    []byte("abcdefghbcdefghicdefghijdefghijkefghijklfghijklmghijklmnhijklmnoijklmnopjklmnopqklmnopqrlmnopqrsmnopqrstnopqrstu"),
			expected: "cf5b16a778af8380036ce59e7b0492370b249b11e8f07a51afac45037afee9d1",
		},
		{
			name:     "one_million_a",
			input:    bytes.Repeat([]byte("a"), 1000000),
			expected: "cdc76e5c9914fb9281a1c7e284d73e67f1809a48a497200e046d39ccc7112cd0",
		},
		{
			name:     "single_byte",
			input:    []byte{0x61}, // 'a'
			expected: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
		},
		{
			name:     "null_byte",
			input:    []byte{0x00},
			expected: "6e340b9cffb37a989ca544e6bb780a2c78901d3fb33738768511a30617afa01d",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := HashSHA256(tc.input)
			resultHex := hex.EncodeToString(result)

			assert.Equal(t, tc.expected, resultHex, "SHA256 hash mismatch for %s", tc.name)
			assert.Len(t, result, 32, "SHA256 should always return 32 bytes")
		})
	}
}

func TestHashSHA256Hex(t *testing.T) {
	t.Parallel()

	input := []byte("test message")

	// Test that hex function matches manual encoding
	hashBytes := HashSHA256(input)
	expectedHex := hex.EncodeToString(hashBytes)

	resultHex := HashSHA256Hex(input)

	assert.Equal(t, expectedHex, resultHex, "HashSHA256Hex should match manual hex encoding")
}

func TestHMACSHA256(t *testing.T) {
	t.Parallel()

	// Test vectors from RFC 4231
	tests := []struct {
		name     string
		key      string // hex encoded
		data     string // hex encoded
		expected string // hex encoded
	}{
		{
			name:     "rfc4231_test_1",
			key:      "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b",
			data:     "4869205468657265", // "Hi There"
			expected: "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7",
		},
		{
			name:     "rfc4231_test_2",
			key:      "4a656665",                                                 // "Jefe"
			data:     "7768617420646f2079612077616e7420666f72206e6f7468696e673f", // "what do ya want for nothing?"
			expected: "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843",
		},
		{
			name:     "rfc4231_test_3",
			key:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			data:     "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			expected: "773ea91e36800e46854db8ebd09181a72959098b3ef8c122d9635514ced565fe",
		},
		{
			name:     "rfc4231_test_4",
			key:      "0102030405060708090a0b0c0d0e0f10111213141516171819",
			data:     "cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd",
			expected: "82558a389a443c0ea4cc819899f2083a85f0faa3e578f8077a2e3ff46729665b",
		},
		{
			name:     "rfc4231_test_6",
			key:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			data:     "54657374205573696e67204c6172676572205468616e20426c6f636b2d53697a65204b6579202d2048617368204b6579204669727374", // "Test Using Larger Than Block-Size Key - Hash Key First"
			expected: "60e431591ee0b67f0d8a26aacbf5b77f8e0bc6213728c5140546040f0ee37f54",
		},
		{
			name:     "empty_data",
			key:      "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b",
			data:     "",
			expected: "999a901219f032cd497cadb5e6051e97b6a29ab297bd6ae722bd6062a2f59542",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			key, err := hex.DecodeString(tc.key)
			require.NoError(t, err, "failed to decode key")

			data, err := hex.DecodeString(tc.data)
			require.NoError(t, err, "failed to decode data")

			result := HMACSHA256(key, data)
			resultHex := hex.EncodeToString(result)

			assert.Equal(t, tc.expected, resultHex, "HMAC-SHA256 mismatch for %s", tc.name)
			assert.Len(t, result, 32, "HMAC-SHA256 should always return 32 bytes")
		})
	}
}

func TestHMACSHA256Hex(t *testing.T) {
	t.Parallel()

	key := []byte("test key")
	data := []byte("test data")

	// Test that hex function matches manual encoding
	hmacBytes := HMACSHA256(key, data)
	expectedHex := hex.EncodeToString(hmacBytes)

	resultHex := HMACSHA256Hex(key, data)

	assert.Equal(t, expectedHex, resultHex, "HMACSHA256Hex should match manual hex encoding")
}

func TestHashSHA256ArrayParity(t *testing.T) {
	input := []byte("hello world")
	slice := HashSHA256(input)
	arr := HashSHA256Array(input)
	assert.Equal(t, slice, arr[:], "slice and array variants should produce identical output")
}

func TestHMACSHA256ArrayParity(t *testing.T) {
	key := []byte("secret-key")
	data := []byte("payload")
	slice := HMACSHA256(key, data)
	arr := HMACSHA256Array(key, data)
	assert.Equal(t, slice, arr[:], "slice and array HMAC variants should produce identical output")
}

func FuzzVerifyHMACSHA256(f *testing.F) {
	// Seed corpus
	f.Add([]byte("k"), []byte("d"))
	f.Add([]byte("another-key"), []byte("some data"))

	f.Fuzz(func(t *testing.T, key, data []byte) {
		mac := HMACSHA256(key, data)
		if !VerifyHMACSHA256(key, data, mac) {
			t.Fatalf("verification failed for valid mac")
		}
		// Tamper and ensure verification fails
		tampered := make([]byte, len(mac))
		copy(tampered, mac)
		if len(tampered) > 0 {
			tampered[0] ^= 0x01
			if VerifyHMACSHA256(key, data, tampered) {
				t.Fatalf("verification succeeded for tampered mac")
			}
		}
	})
}

func TestVerifyHMACSHA256(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		key         []byte
		data        []byte
		expectedMAC []byte
		shouldMatch bool
	}{
		{
			name:        "valid_mac",
			key:         []byte("secret-key"),
			data:        []byte("message"),
			expectedMAC: nil, // will be computed
			shouldMatch: true,
		},
		{
			name:        "invalid_mac_wrong_key",
			key:         []byte("wrong-key"),
			data:        []byte("message"),
			expectedMAC: nil, // will be computed with different key
			shouldMatch: false,
		},
		{
			name:        "invalid_mac_wrong_data",
			key:         []byte("secret-key"),
			data:        []byte("different-message"),
			expectedMAC: nil, // will be computed with different data
			shouldMatch: false,
		},
		{
			name:        "invalid_mac_tampered",
			key:         []byte("secret-key"),
			data:        []byte("message"),
			expectedMAC: []byte("completely-wrong-mac-value-here"),
			shouldMatch: false,
		},
		{
			name:        "empty_mac",
			key:         []byte("secret-key"),
			data:        []byte("message"),
			expectedMAC: []byte{},
			shouldMatch: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Compute expected MAC if not provided
			if tc.expectedMAC == nil {
				if tc.shouldMatch {
					tc.expectedMAC = HMACSHA256(tc.key, tc.data)
				} else {
					// Generate MAC with different inputs for negative tests
					if tc.name == "invalid_mac_wrong_key" {
						tc.expectedMAC = HMACSHA256([]byte("secret-key"), tc.data)
					} else if tc.name == "invalid_mac_wrong_data" {
						tc.expectedMAC = HMACSHA256(tc.key, []byte("message"))
					}
				}
			}

			result := VerifyHMACSHA256(tc.key, tc.data, tc.expectedMAC)

			if tc.shouldMatch {
				assert.True(t, result, "HMAC verification should succeed for %s", tc.name)
			} else {
				assert.False(t, result, "HMAC verification should fail for %s", tc.name)
			}
		})
	}
}

func TestSecureCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		a     []byte
		b     []byte
		equal bool
	}{
		{
			name:  "equal_empty",
			a:     []byte{},
			b:     []byte{},
			equal: true,
		},
		{
			name:  "equal_single_byte",
			a:     []byte{0x42},
			b:     []byte{0x42},
			equal: true,
		},
		{
			name:  "equal_multiple_bytes",
			a:     []byte("test data"),
			b:     []byte("test data"),
			equal: true,
		},
		{
			name:  "equal_binary_data",
			a:     []byte{0x00, 0x01, 0x02, 0x03, 0xff},
			b:     []byte{0x00, 0x01, 0x02, 0x03, 0xff},
			equal: true,
		},
		{
			name:  "not_equal_different_length",
			a:     []byte("short"),
			b:     []byte("longer string"),
			equal: false,
		},
		{
			name:  "not_equal_same_length",
			a:     []byte("test"),
			b:     []byte("best"),
			equal: false,
		},
		{
			name:  "not_equal_one_bit_difference",
			a:     []byte{0x00},
			b:     []byte{0x01},
			equal: false,
		},
		{
			name:  "not_equal_last_byte_different",
			a:     []byte("test1"),
			b:     []byte("test2"),
			equal: false,
		},
		{
			name:  "not_equal_first_byte_different",
			a:     []byte("1test"),
			b:     []byte("2test"),
			equal: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := SecureCompare(tc.a, tc.b)

			if tc.equal {
				assert.True(t, result, "SecureCompare should return true for %s", tc.name)
			} else {
				assert.False(t, result, "SecureCompare should return false for %s", tc.name)
			}
		})
	}
}

func TestSecureCompareStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		a     string
		b     string
		equal bool
	}{
		{
			name:  "equal_empty",
			a:     "",
			b:     "",
			equal: true,
		},
		{
			name:  "equal_strings",
			a:     "password123",
			b:     "password123",
			equal: true,
		},
		{
			name:  "not_equal",
			a:     "password123",
			b:     "password124",
			equal: false,
		},
		{
			name:  "unicode_equal",
			a:     "пароль",
			b:     "пароль",
			equal: true,
		},
		{
			name:  "unicode_not_equal",
			a:     "пароль",
			b:     "парол",
			equal: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := SecureCompareStrings(tc.a, tc.b)

			if tc.equal {
				assert.True(t, result, "SecureCompareStrings should return true for %s", tc.name)
			} else {
				assert.False(t, result, "SecureCompareStrings should return false for %s", tc.name)
			}
		})
	}
}

func TestConstantTimeComparison(t *testing.T) {
	t.Parallel()

	// Test that comparison time is constant regardless of where the difference is
	// This is a basic timing test - more sophisticated timing analysis would be needed
	// for cryptographic validation

	data1 := bytes.Repeat([]byte("a"), 1000)
	data2 := bytes.Repeat([]byte("a"), 1000)
	data3 := bytes.Repeat([]byte("a"), 1000)

	// Modify at different positions
	data2[0] = 'b'   // First byte different
	data3[999] = 'b' // Last byte different

	// Measure timing for comparisons
	iterations := 10000

	// Equal comparison
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_ = SecureCompare(data1, data1)
	}
	equalTime := time.Since(start)

	// First byte different
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_ = SecureCompare(data1, data2)
	}
	firstDiffTime := time.Since(start)

	// Last byte different
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_ = SecureCompare(data1, data3)
	}
	lastDiffTime := time.Since(start)

	// Calculate ratios - should be close to 1.0 for constant time
	firstRatio := float64(firstDiffTime) / float64(equalTime)
	lastRatio := float64(lastDiffTime) / float64(equalTime)

	// These should be relatively close for constant-time comparison
	// Allow for some variance due to system noise
	assert.InDelta(t, 1.0, firstRatio, 0.5, "First byte difference timing should be similar")
	assert.InDelta(t, 1.0, lastRatio, 0.5, "Last byte difference timing should be similar")
	assert.InDelta(t, firstRatio, lastRatio, 0.3, "Different position timings should be similar")
}

func TestHashConcurrency(t *testing.T) {
	t.Parallel()

	// Test that hash functions are safe for concurrent use
	data := []byte("concurrent test data")
	key := []byte("concurrent key")

	expectedHash := HashSHA256(data)
	expectedHMAC := HMACSHA256(key, data)

	// Run concurrent hash operations
	done := make(chan bool, 4)

	// SHA256 concurrent calls
	go func() {
		for i := 0; i < 100; i++ {
			result := HashSHA256(data)
			assert.Equal(t, expectedHash, result, "SHA256 mismatch at iteration %d", i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			result := HashSHA256(data)
			assert.Equal(t, expectedHash, result, "SHA256 mismatch at iteration %d", i)
		}
		done <- true
	}()

	// HMAC concurrent calls
	go func() {
		for i := 0; i < 100; i++ {
			result := HMACSHA256(key, data)
			assert.Equal(t, expectedHMAC, result, "HMAC mismatch at iteration %d", i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			result := HMACSHA256(key, data)
			assert.Equal(t, expectedHMAC, result, "HMAC mismatch at iteration %d", i)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}

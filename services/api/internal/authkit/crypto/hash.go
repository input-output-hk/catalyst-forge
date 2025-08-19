// Utilities for hashing and MACs used by AuthKit.
//
// Security note: these primitives are NOT suitable for password hashing.
// For passwords, use dedicated KDFs such as bcrypt, scrypt, or argon2id.
// For message authentication, prefer HMAC interfaces and constant-time
// comparisons. See function docs for allocation characteristics where relevant.
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// HashSHA256 computes the SHA256 hash of the input.
func HashSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashSHA256Array computes the SHA256 hash and returns a fixed-size array to
// avoid an allocation. Prefer this in hot paths when a slice is not required.
func HashSHA256Array(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// HashSHA256Hex computes the SHA256 hash and returns it as a hex string.
func HashSHA256Hex(data []byte) string {
	return hex.EncodeToString(HashSHA256(data))
}

// HMACSHA256 computes the HMAC-SHA256 of the data with the given key.
func HMACSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// HMACSHA256Array computes the HMAC-SHA256 and returns a fixed-size array to
// avoid an allocation. Prefer this in hot paths when a slice is not required.
func HMACSHA256Array(key, data []byte) [32]byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(data)
	var out [32]byte
	// Write sum directly into preallocated array-backed slice to avoid allocs.
	h.Sum(out[:0])
	return out
}

// HMACSHA256Hex computes the HMAC-SHA256 and returns it as a hex string.
func HMACSHA256Hex(key, data []byte) string {
	return hex.EncodeToString(HMACSHA256(key, data))
}

// VerifyHMACSHA256 verifies that the expected HMAC matches the computed HMAC.
//
// Uses constant-time comparison to prevent timing attacks.
func VerifyHMACSHA256(key, data, expectedMAC []byte) bool {
	mac := HMACSHA256(key, data)
	return hmac.Equal(mac, expectedMAC)
}

// SecureCompare performs a constant-time comparison of two byte slices.
func SecureCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

// SecureCompareStrings performs a constant-time comparison of two strings.
//
// Note: this converts strings to byte slices and allocates. Prefer
// SecureCompare([]byte, []byte) for secrets to avoid allocations and reduce
// lifetime of sensitive material on the heap.
func SecureCompareStrings(a, b string) bool {
	return SecureCompare([]byte(a), []byte(b))
}

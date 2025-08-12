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
func SecureCompareStrings(a, b string) bool {
	return SecureCompare([]byte(a), []byte(b))
}
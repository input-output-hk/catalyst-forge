//go:build test
// +build test

package cache

// MustCanonicalize canonicalizes JSON data or panics on error.
// Test-only helper.
func MustCanonicalize(data []byte) []byte {
	result, err := Canonicalize(data)
	if err != nil {
		panic(err)
	}
	return result
}

// MustHash computes the hash of canonical JSON or panics on error.
// Test-only helper.
func MustHash(data []byte) string {
	hash, err := Hash(data)
	if err != nil {
		panic(err)
	}
	return hash
}

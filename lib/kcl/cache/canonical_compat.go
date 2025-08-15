package cache

// CanonicalizeJSON is kept for backward compatibility (tests and callers).
// Prefer using Canonicalize.
func CanonicalizeJSON(data []byte) ([]byte, error) {
	return Canonicalize(data)
}
